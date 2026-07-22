// Package telegram wires the go-telegram/bot client to the application's
// handlers: registering commands, the lobby-creation wizard, and the
// join/approval callback flows.
package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"

	"meet-up-bot/internal/i18n"
	"meet-up-bot/internal/storage"
	"meet-up-bot/internal/storage/db"
)

// botClient aliases the go-telegram client so handler signatures stay concise
// and the individual handler files don't each need to import the bot package.
type botClient = bot.Bot

// Bot bundles the telegram client with its dependencies.
type Bot struct {
	api      *bot.Bot
	store    *storage.Store
	log      *zap.Logger
	sessions *sessions
	username string // the bot's @username, used to build invite deep links
}

// New constructs the bot, registers all handlers, and returns it ready to Start.
func New(token string, store *storage.Store, log *zap.Logger) (*Bot, error) {
	b := &Bot{
		store:    store,
		log:      log,
		sessions: newSessions(),
	}

	opts := []bot.Option{
		// The default handler receives any message that no command handler
		// matched — that's where free-text answers to the wizard land.
		bot.WithDefaultHandler(b.onText),
	}

	api, err := bot.New(token, opts...)
	if err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}
	b.api = api

	b.registerHandlers()
	return b, nil
}

func (b *Bot) registerHandlers() {
	// "start" as a command match so it also catches deep links like
	// "/start join_42" from invite URLs.
	b.api.RegisterHandler(bot.HandlerTypeMessageText, "start", bot.MatchTypeCommandStartOnly, b.onStart)
	b.api.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, b.onStart)
	b.api.RegisterHandler(bot.HandlerTypeMessageText, "/create", bot.MatchTypeExact, b.onCreate)
	b.api.RegisterHandler(bot.HandlerTypeMessageText, "/lobbies", bot.MatchTypeExact, b.onListLobbies)
	b.api.RegisterHandler(bot.HandlerTypeMessageText, "/mylobbies", bot.MatchTypeExact, b.onMyLobbies)
	b.api.RegisterHandler(bot.HandlerTypeMessageText, "/settings", bot.MatchTypeExact, b.onSettings)
	b.api.RegisterHandler(bot.HandlerTypeMessageText, "/cancel", bot.MatchTypeExact, b.onCancel)

	// Callback buttons are namespaced by an "action:" prefix.
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbVisibility+":", bot.MatchTypePrefix, b.onVisibility)
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbJoin+":", bot.MatchTypePrefix, b.onJoin)
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbApprove+":", bot.MatchTypePrefix, b.onApprove)
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbReject+":", bot.MatchTypePrefix, b.onReject)
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbMembers+":", bot.MatchTypePrefix, b.onMembers)
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbRemove+":", bot.MatchTypePrefix, b.onRemoveMember)
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbBan+":", bot.MatchTypePrefix, b.onBan)
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbEdit+":", bot.MatchTypePrefix, b.onEdit)
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbEditField+":", bot.MatchTypePrefix, b.onEditField)
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbEditVis+":", bot.MatchTypePrefix, b.onEditVisibility)
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbLocale+":", bot.MatchTypePrefix, b.onLocale)
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbDetails+":", bot.MatchTypePrefix, b.onDetails)
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbSettings+":", bot.MatchTypePrefix, b.onSettingsMenu)
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbTimeFilter+":", bot.MatchTypePrefix, b.onTimeFilter)
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbLobbyPage+":", bot.MatchTypePrefix, b.onLobbyPage)
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbDelete+":", bot.MatchTypePrefix, b.onDeleteLobby)
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbDeleteYes+":", bot.MatchTypePrefix, b.onDeleteConfirm)
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbLeave+":", bot.MatchTypePrefix, b.onLeaveLobby)
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbDismiss+":", bot.MatchTypePrefix, b.onDismiss)
	b.api.RegisterHandler(bot.HandlerTypeCallbackQueryData, cbInvite+":", bot.MatchTypePrefix, b.onInvite)
}

// memberStatus returns the user's membership status for a lobby and whether a
// membership row exists at all.
func (b *Bot) memberStatus(ctx context.Context, lobbyID, userID int64) (db.MembershipStatus, bool) {
	m, err := b.store.GetMember(ctx, db.GetMemberParams{LobbyID: lobbyID, UserID: userID})
	if err != nil {
		return "", false
	}
	return m.Status, true
}

// tr returns a translator in the given user's stored locale, defaulting to
// English when the user is unknown or on error.
func (b *Bot) tr(ctx context.Context, userID int64) *i18n.Translator {
	if u, err := b.store.GetUser(ctx, userID); err == nil {
		return i18n.For(i18n.Parse(u.Locale))
	}
	return i18n.For(i18n.En)
}

// userTimezone returns the user's stored IANA timezone name, or "UTC".
func (b *Bot) userTimezone(ctx context.Context, userID int64) string {
	if u, err := b.store.GetUser(ctx, userID); err == nil && u.Timezone != "" {
		return u.Timezone
	}
	return "UTC"
}

// viewer returns both the translator and timezone for a user in a single
// lookup, defaulting to English/UTC on error. Use it wherever a lobby is
// rendered (its time is shown in the viewer's timezone).
func (b *Bot) viewer(ctx context.Context, userID int64) (*i18n.Translator, *time.Location) {
	if u, err := b.store.GetUser(ctx, userID); err == nil {
		return i18n.For(i18n.Parse(u.Locale)), loadLocation(u.Timezone)
	}
	return i18n.For(i18n.En), time.UTC
}

// Start runs the long-polling loop until ctx is cancelled.
func (b *Bot) Start(ctx context.Context) {
	// Resolve our own @username so invite deep links can be built.
	if me, err := b.api.GetMe(ctx); err == nil {
		b.username = me.Username
		b.log.Info("telegram bot identified", zap.String("username", me.Username))
	} else {
		b.log.Warn("could not resolve bot username; invite links unavailable", zap.Error(err))
	}
	b.log.Info("starting telegram long polling")
	b.api.Start(ctx)
}

// send is a small helper around SendMessage that logs failures instead of
// bubbling them up (there is rarely anything a handler can do about them).
func (b *Bot) send(ctx context.Context, chatID int64, text string, markup models.ReplyMarkup) {
	_, err := b.api.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: markup,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: ptr(true),
		},
	})
	if err != nil {
		b.log.Error("send message failed", zap.Int64("chat_id", chatID), zap.Error(err))
	}
}

// answer acknowledges a callback query (removes the client-side spinner).
func (b *Bot) answer(ctx context.Context, queryID, text string) {
	_, err := b.api.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: queryID,
		Text:            text,
	})
	if err != nil {
		b.log.Warn("answer callback failed", zap.Error(err))
	}
}

func ptr[T any](v T) *T { return &v }

package telegram

import (
	"context"
	"strings"

	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"

	"meet-up-bot/internal/i18n"
	"meet-up-bot/internal/storage/db"
)

// onCreate starts the lobby-creation wizard.
func (b *Bot) onCreate(ctx context.Context, _ *botClient, update *models.Update) {
	msg := update.Message
	if msg == nil || msg.From == nil {
		return
	}
	user, err := b.ensureUser(ctx, msg.From)
	if err != nil {
		b.log.Error("ensure user", zap.Error(err))
		b.send(ctx, msg.Chat.ID, b.tr(ctx, msg.From.ID).T(i18n.KeyErrGeneric), nil)
		return
	}
	tr := i18n.For(i18n.Parse(user.Locale))
	b.sessions.startCreate(msg.From.ID)
	b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyCreateStart), nil)
}

// onText is the default handler: it advances the active conversation (create
// wizard or edit input) when the user has one, and otherwise nudges them toward
// the commands.
func (b *Bot) onText(ctx context.Context, _ *botClient, update *models.Update) {
	msg := update.Message
	if msg == nil || msg.From == nil {
		return
	}
	tr := b.tr(ctx, msg.From.ID)

	s := b.sessions.get(msg.From.ID)
	if s == nil {
		b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyDidntGet), nil)
		return
	}
	switch s.kind {
	case kindEdit:
		b.handleEditInput(ctx, tr, msg, s)
		return
	case kindSettings:
		b.handleSettingsInput(ctx, tr, msg, s)
		return
	}

	loc := loadLocation(b.userTimezone(ctx, msg.From.ID))
	text := strings.TrimSpace(msg.Text)
	switch s.step {
	case stepName:
		if text == "" {
			b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyNameEmpty), nil)
			return
		}
		s.name = text
		s.step = stepCountry
		b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyAskCountry), nil)

	case stepCountry:
		if text == "" {
			b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyCountryEmpty), nil)
			return
		}
		s.country = text
		s.step = stepCity
		b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyAskCity), nil)

	case stepCity:
		if text == "" {
			b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyCityEmpty), nil)
			return
		}
		s.city = text
		s.step = stepAddress
		b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyAskAddress), nil)

	case stepAddress:
		if text != "-" && text != "" {
			s.address = text
		}
		s.step = stepTime
		b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyAskTime, displayTime, loc.String()), nil)

	case stepTime:
		t, err := parseEventTimeIn(text, loc)
		if err != nil {
			b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyTimeBad, displayTime), nil)
			return
		}
		s.eventTime = t
		s.step = stepChatLink
		b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyAskLink), nil)

	case stepChatLink:
		if text != "-" && text != "" {
			s.chatLink = text
		}
		s.step = stepVisibility
		b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyAskVisibility), visibilityKeyboard(tr))

	default:
		b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyPickButtons), nil)
	}
}

func visibilityKeyboard(tr *i18n.Translator) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{{
			{Text: tr.T(i18n.KeyBtnPublic), CallbackData: cbVisibility + ":" + string(db.LobbyVisibilityPublic)},
			{Text: tr.T(i18n.KeyBtnPrivateApproval), CallbackData: cbVisibility + ":" + string(db.LobbyVisibilityPrivate)},
		}},
	}
}

// onVisibility handles the final wizard step and persists the lobby.
func (b *Bot) onVisibility(ctx context.Context, _ *botClient, update *models.Update) {
	q := update.CallbackQuery
	if q == nil || q.Message.Message == nil {
		return
	}
	chatID := q.Message.Message.Chat.ID
	userID := q.From.ID
	tr, loc := b.viewer(ctx, userID)

	s := b.sessions.get(userID)
	if s == nil || s.kind != kindCreate || s.step != stepVisibility {
		b.answer(ctx, q.ID, tr.T(i18n.KeyWizardExpired))
		return
	}

	_, value, _ := strings.Cut(q.Data, ":")
	switch db.LobbyVisibility(value) {
	case db.LobbyVisibilityPublic, db.LobbyVisibilityPrivate:
		s.visibility = db.LobbyVisibility(value)
	default:
		b.answer(ctx, q.ID, tr.T(i18n.KeyUnknownOption))
		return
	}
	b.answer(ctx, q.ID, "")

	lobby, err := b.store.CreateLobby(ctx, db.CreateLobbyParams{
		CreatorID:  userID,
		Name:       s.name,
		Country:    s.country,
		City:       s.city,
		Address:    nullableText(s.address),
		EventTime:  toTimestamptz(s.eventTime),
		ChatLink:   nullableText(s.chatLink),
		Visibility: s.visibility,
	})
	if err != nil {
		b.log.Error("create lobby", zap.Error(err))
		b.send(ctx, chatID, tr.T(i18n.KeyErrSaveLobby), nil)
		return
	}

	// The creator is automatically an approved member of their own lobby.
	if _, err := b.store.AddMember(ctx, db.AddMemberParams{
		LobbyID: lobby.ID,
		UserID:  userID,
		Status:  db.MembershipStatusApproved,
	}); err != nil {
		b.log.Error("add creator as member", zap.Error(err))
	}

	b.sessions.clear(userID)
	b.send(ctx, chatID, tr.T(i18n.KeyLobbyCreated)+"\n\n"+formatLobby(tr, loc, lobby, 1, true), nil)
	if lobby.Visibility == db.LobbyVisibilityPublic {
		b.send(ctx, chatID, tr.T(i18n.KeyCreatedPublicHint), nil)
	} else {
		b.send(ctx, chatID, tr.T(i18n.KeyCreatedPrivateHint), nil)
	}
}

package telegram

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"meet-up-bot/internal/i18n"
	"meet-up-bot/internal/storage/db"
)

// ensureUser upserts the telegram user so foreign keys resolve, returning the
// stored row.
func (b *Bot) ensureUser(ctx context.Context, u *models.User) (db.User, error) {
	return b.store.UpsertUser(ctx, db.UpsertUserParams{
		ID:        u.ID,
		Username:  nullableText(u.Username),
		FirstName: u.FirstName,
		LastName:  u.LastName,
	})
}

func (b *Bot) onStart(ctx context.Context, _ *botClient, update *models.Update) {
	msg := update.Message
	if msg == nil || msg.From == nil {
		return
	}
	user, err := b.ensureUser(ctx, msg.From)
	if err != nil {
		b.log.Error("ensure user", zap.Error(err))
	}
	tr := i18n.For(i18n.Parse(user.Locale))
	b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyHelp), nil)
}

func (b *Bot) onCancel(ctx context.Context, _ *botClient, update *models.Update) {
	msg := update.Message
	if msg == nil || msg.From == nil {
		return
	}
	b.sessions.clear(msg.From.ID)
	b.send(ctx, msg.Chat.ID, b.tr(ctx, msg.From.ID).T(i18n.KeyCancelled), nil)
}

// onListLobbies shows the first page of upcoming lobbies, filtered by the
// user's settings (city + time window).
func (b *Bot) onListLobbies(ctx context.Context, _ *botClient, update *models.Update) {
	msg := update.Message
	if msg == nil || msg.From == nil {
		return
	}
	if _, err := b.ensureUser(ctx, msg.From); err != nil {
		b.log.Error("ensure user", zap.Error(err))
	}
	b.sendLobbiesPage(ctx, msg.Chat.ID, msg.From.ID, 0)
}

// onMyLobbies lists every lobby the caller takes part in — as admin (creator)
// or as a member — as a compact list. Each row carries a Details button that
// opens the full card and role-appropriate actions.
func (b *Bot) onMyLobbies(ctx context.Context, _ *botClient, update *models.Update) {
	msg := update.Message
	if msg == nil || msg.From == nil {
		return
	}
	if _, err := b.ensureUser(ctx, msg.From); err != nil {
		b.log.Error("ensure user", zap.Error(err))
	}
	tr := b.tr(ctx, msg.From.ID)

	lobbies, err := b.store.ListMyLobbies(ctx, msg.From.ID)
	if err != nil {
		b.log.Error("list my lobbies", zap.Error(err))
		b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyErrLoadYourLobbies), nil)
		return
	}
	if len(lobbies) == 0 {
		b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyNoMyLobbies), nil)
		return
	}

	for _, l := range lobbies {
		line := fmt.Sprintf("<b>%s</b> — %s", htmlEscape(l.Name), roleLabel(tr, l, msg.From.ID))
		markup := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{{
				{Text: tr.T(i18n.KeyBtnDetails), CallbackData: fmt.Sprintf("%s:%d", cbDetails, l.ID)},
			}},
		}
		b.send(ctx, msg.Chat.ID, line, markup)
	}
}

// roleLabel returns the localized admin/member badge for the user in a lobby.
func roleLabel(tr *i18n.Translator, l db.Lobby, userID int64) string {
	if l.CreatorID == userID {
		return tr.T(i18n.KeyRoleAdmin)
	}
	return tr.T(i18n.KeyRoleMember)
}

// onDetails shows the full card for one of the user's lobbies plus the actions
// available to them: admins get Manage-members, Edit, and pending requests;
// members just see the details (including the chat link).
func (b *Bot) onDetails(ctx context.Context, _ *botClient, update *models.Update) {
	q := update.CallbackQuery
	if q == nil {
		return
	}
	tr, loc := b.viewer(ctx, q.From.ID)

	lobbyID, ok := b.parseSingleID(ctx, q, cbDetails)
	if !ok {
		return
	}
	lobby, err := b.store.GetLobby(ctx, lobbyID)
	if err != nil {
		b.answer(ctx, q.ID, tr.T(i18n.KeyLobbyGone))
		return
	}

	uid := q.From.ID
	isAdmin := lobby.CreatorID == uid
	status, isMember := b.memberStatus(ctx, lobbyID, uid)
	if !isAdmin && !(isMember && status == db.MembershipStatusApproved) {
		b.answer(ctx, q.ID, tr.T(i18n.KeyNotAMember))
		return
	}
	b.answer(ctx, q.ID, "")

	count, _ := b.store.CountApprovedMembers(ctx, lobbyID)
	text := formatLobby(tr, loc, lobby, count, true) + "\n" + roleLabel(tr, lobby, uid)

	if !isAdmin {
		// Members get a Leave button.
		markup := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{{
				{Text: tr.T(i18n.KeyBtnLeave), CallbackData: fmt.Sprintf("%s:%d", cbLeave, lobby.ID)},
			}},
		}
		b.send(ctx, uid, text, markup)
		return
	}

	markup := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: tr.T(i18n.KeyBtnManageMembers), CallbackData: fmt.Sprintf("%s:%d", cbMembers, lobby.ID)},
				{Text: tr.T(i18n.KeyBtnEdit), CallbackData: fmt.Sprintf("%s:%d", cbEdit, lobby.ID)},
			},
			{
				{Text: tr.T(i18n.KeyBtnDeleteLobby), CallbackData: fmt.Sprintf("%s:%d", cbDelete, lobby.ID)},
			},
		},
	}
	b.send(ctx, uid, text, markup)

	if lobby.Visibility == db.LobbyVisibilityPrivate {
		if err := b.sendPendingRequests(ctx, tr, uid, lobby); err != nil {
			b.log.Error("send pending requests", zap.Error(err))
		}
	}
}

// sendPendingRequests posts one approve/reject prompt per pending join request.
func (b *Bot) sendPendingRequests(ctx context.Context, tr *i18n.Translator, chatID int64, l db.Lobby) error {
	pending, err := b.store.ListPendingMembers(ctx, l.ID)
	if err != nil {
		return err
	}
	for _, m := range pending {
		who := fmt.Sprintf("User %d", m.UserID)
		if u, err := b.store.GetUser(ctx, m.UserID); err == nil {
			who = describeUser(u)
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		markup := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{{
				{Text: tr.T(i18n.KeyBtnApprove), CallbackData: fmt.Sprintf("%s:%d:%d", cbApprove, l.ID, m.UserID)},
				{Text: tr.T(i18n.KeyBtnReject), CallbackData: fmt.Sprintf("%s:%d:%d", cbReject, l.ID, m.UserID)},
			}},
		}
		b.send(ctx, chatID, tr.T(i18n.KeyPendingRequest, htmlEscape(l.Name), who), markup)
	}
	return nil
}

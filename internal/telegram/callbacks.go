package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"meet-up-bot/internal/i18n"
	"meet-up-bot/internal/storage/db"
)

// onJoin handles the "Join" button on a public/private lobby.
func (b *Bot) onJoin(ctx context.Context, _ *botClient, update *models.Update) {
	q := update.CallbackQuery
	if q == nil {
		return
	}
	if _, err := b.ensureUser(ctx, &q.From); err != nil {
		b.log.Error("ensure user on join", zap.Error(err))
	}
	tr := b.tr(ctx, q.From.ID)

	lobbyID, ok := b.parseSingleID(ctx, q, cbJoin)
	if !ok {
		return
	}

	lobby, err := b.store.GetLobby(ctx, lobbyID)
	if errors.Is(err, pgx.ErrNoRows) {
		b.answer(ctx, q.ID, tr.T(i18n.KeyLobbyGone))
		return
	}
	if err != nil {
		b.log.Error("get lobby on join", zap.Error(err))
		b.answer(ctx, q.ID, tr.T(i18n.KeyErrGeneric))
		return
	}

	// Already a member? Report the current state instead of re-adding.
	if existing, err := b.store.GetMember(ctx, db.GetMemberParams{LobbyID: lobbyID, UserID: q.From.ID}); err == nil {
		switch existing.Status {
		case db.MembershipStatusApproved:
			b.answer(ctx, q.ID, tr.T(i18n.KeyAlreadyIn))
		case db.MembershipStatusPending:
			b.answer(ctx, q.ID, tr.T(i18n.KeyStillPending))
		case db.MembershipStatusRejected:
			b.answer(ctx, q.ID, tr.T(i18n.KeyWasDeclined))
		case db.MembershipStatusBanned:
			b.answer(ctx, q.ID, tr.T(i18n.KeyBanned))
		}
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		b.log.Error("get member on join", zap.Error(err))
		b.answer(ctx, q.ID, tr.T(i18n.KeyErrGeneric))
		return
	}

	if lobby.Visibility == db.LobbyVisibilityPublic {
		b.joinPublic(ctx, tr, q, lobby)
		return
	}
	b.joinPrivate(ctx, tr, q, lobby)
}

func (b *Bot) joinPublic(ctx context.Context, tr *i18n.Translator, q *models.CallbackQuery, lobby db.Lobby) {
	if _, err := b.store.AddMember(ctx, db.AddMemberParams{
		LobbyID: lobby.ID,
		UserID:  q.From.ID,
		Status:  db.MembershipStatusApproved,
	}); err != nil {
		b.log.Error("add member public", zap.Error(err))
		b.answer(ctx, q.ID, tr.T(i18n.KeyErrGeneric))
		return
	}
	b.answer(ctx, q.ID, tr.T(i18n.KeyJoinedToast))
	b.send(ctx, q.From.ID, tr.T(i18n.KeyJoinedMsg, htmlEscape(lobby.Name)), nil)
	if lobby.ChatLink != nil && *lobby.ChatLink != "" {
		b.send(ctx, q.From.ID, tr.T(i18n.KeyChatLine, htmlEscape(*lobby.ChatLink)), nil)
	}
}

func (b *Bot) joinPrivate(ctx context.Context, tr *i18n.Translator, q *models.CallbackQuery, lobby db.Lobby) {
	if _, err := b.store.AddMember(ctx, db.AddMemberParams{
		LobbyID: lobby.ID,
		UserID:  q.From.ID,
		Status:  db.MembershipStatusPending,
	}); err != nil {
		b.log.Error("add member private", zap.Error(err))
		b.answer(ctx, q.ID, tr.T(i18n.KeyErrGeneric))
		return
	}
	b.answer(ctx, q.ID, tr.T(i18n.KeyRequestSentToast))
	b.send(ctx, q.From.ID, tr.T(i18n.KeyRequestSentMsg, htmlEscape(lobby.Name)), nil)

	// Notify the creator (in their own locale) with approve/reject buttons.
	b.notifyRequestToAdmin(ctx, lobby, q.From)
}

// onApprove and onReject are the admin decisions on a pending request.
func (b *Bot) onApprove(ctx context.Context, _ *botClient, update *models.Update) {
	b.decide(ctx, update, db.MembershipStatusApproved)
}

func (b *Bot) onReject(ctx context.Context, _ *botClient, update *models.Update) {
	b.decide(ctx, update, db.MembershipStatusRejected)
}

func (b *Bot) decide(ctx context.Context, update *models.Update, decision db.MembershipStatus) {
	q := update.CallbackQuery
	if q == nil {
		return
	}
	tr := b.tr(ctx, q.From.ID)
	action := cbApprove
	if decision == db.MembershipStatusRejected {
		action = cbReject
	}

	lobbyID, applicantID, ok := b.parseTwoIDs(ctx, q, action)
	if !ok {
		return
	}

	lobby, err := b.store.GetLobby(ctx, lobbyID)
	if err != nil {
		b.answer(ctx, q.ID, tr.T(i18n.KeyLobbyGone))
		return
	}
	if lobby.CreatorID != q.From.ID {
		b.answer(ctx, q.ID, tr.T(i18n.KeyAdminOnly))
		return
	}

	if _, err := b.store.UpdateMemberStatus(ctx, db.UpdateMemberStatusParams{
		LobbyID: lobbyID,
		UserID:  applicantID,
		Status:  decision,
	}); err != nil {
		b.log.Error("update member status", zap.Error(err))
		b.answer(ctx, q.ID, tr.T(i18n.KeyErrGeneric))
		return
	}

	// The applicant is notified in their own locale.
	applicantTr := b.tr(ctx, applicantID)
	if decision == db.MembershipStatusApproved {
		b.answer(ctx, q.ID, tr.T(i18n.KeyApprovedToast))
		b.editDecision(ctx, q, tr.T(i18n.KeyApprovedEdit, htmlEscape(lobby.Name)))
		b.send(ctx, applicantID, applicantTr.T(i18n.KeyApprovedMsg, htmlEscape(lobby.Name)), nil)
		if lobby.ChatLink != nil && *lobby.ChatLink != "" {
			b.send(ctx, applicantID, applicantTr.T(i18n.KeyChatLine, htmlEscape(*lobby.ChatLink)), nil)
		}
	} else {
		b.answer(ctx, q.ID, tr.T(i18n.KeyRejectedToast))
		b.editDecision(ctx, q, tr.T(i18n.KeyRejectedEdit, htmlEscape(lobby.Name)))
		b.send(ctx, applicantID, applicantTr.T(i18n.KeyRejectedMsg, htmlEscape(lobby.Name)), nil)
	}
}

// onMembers lists a lobby's approved members (each with Remove/Ban buttons) so
// the admin can review and kick people. Only the lobby creator may use it.
func (b *Bot) onMembers(ctx context.Context, _ *botClient, update *models.Update) {
	q := update.CallbackQuery
	if q == nil {
		return
	}
	tr := b.tr(ctx, q.From.ID)

	lobbyID, ok := b.parseSingleID(ctx, q, cbMembers)
	if !ok {
		return
	}

	lobby, err := b.store.GetLobby(ctx, lobbyID)
	if err != nil {
		b.answer(ctx, q.ID, tr.T(i18n.KeyLobbyGone))
		return
	}
	if lobby.CreatorID != q.From.ID {
		b.answer(ctx, q.ID, tr.T(i18n.KeyAdminOnly))
		return
	}
	b.answer(ctx, q.ID, "")

	members, err := b.store.ListApprovedMembers(ctx, lobbyID)
	if err != nil {
		b.log.Error("list approved members", zap.Error(err))
		b.send(ctx, q.From.ID, tr.T(i18n.KeyErrLoadMembers), nil)
		return
	}

	var shown int
	for _, m := range members {
		if m.UserID == lobby.CreatorID {
			continue // the admin is a member of their own lobby; don't list them
		}
		shown++
		who := fmt.Sprintf("User %d", m.UserID)
		if u, err := b.store.GetUser(ctx, m.UserID); err == nil {
			who = describeUser(u)
		}
		markup := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{{
				{Text: tr.T(i18n.KeyBtnRemove), CallbackData: fmt.Sprintf("%s:%d:%d", cbRemove, lobbyID, m.UserID)},
				{Text: tr.T(i18n.KeyBtnBan), CallbackData: fmt.Sprintf("%s:%d:%d", cbBan, lobbyID, m.UserID)},
			}},
		}
		b.send(ctx, q.From.ID, fmt.Sprintf("👤 %s", who), markup)
	}

	if shown == 0 {
		b.send(ctx, q.From.ID, tr.T(i18n.KeyNoMembersYet, htmlEscape(lobby.Name)), nil)
	}
}

// onRemoveMember kicks an approved member. Only the lobby creator may do it, and
// they cannot remove themselves.
func (b *Bot) onRemoveMember(ctx context.Context, _ *botClient, update *models.Update) {
	q := update.CallbackQuery
	if q == nil {
		return
	}
	tr := b.tr(ctx, q.From.ID)

	lobbyID, memberID, ok := b.parseTwoIDs(ctx, q, cbRemove)
	if !ok {
		return
	}

	lobby, err := b.store.GetLobby(ctx, lobbyID)
	if err != nil {
		b.answer(ctx, q.ID, tr.T(i18n.KeyLobbyGone))
		return
	}
	if lobby.CreatorID != q.From.ID {
		b.answer(ctx, q.ID, tr.T(i18n.KeyAdminOnly))
		return
	}
	if memberID == lobby.CreatorID {
		b.answer(ctx, q.ID, tr.T(i18n.KeyCantRemoveSelf))
		return
	}

	if err := b.store.RemoveMember(ctx, db.RemoveMemberParams{LobbyID: lobbyID, UserID: memberID}); err != nil {
		b.log.Error("remove member", zap.Error(err))
		b.answer(ctx, q.ID, tr.T(i18n.KeyErrGeneric))
		return
	}

	b.answer(ctx, q.ID, tr.T(i18n.KeyRemovedToast))
	b.editDecision(ctx, q, tr.T(i18n.KeyRemovedEdit, htmlEscape(lobby.Name)))
	b.send(ctx, memberID, b.tr(ctx, memberID).T(i18n.KeyRemovedMsg, htmlEscape(lobby.Name)), nil)
}

// onBan bans a member: unlike removal, it keeps a "banned" row so the user
// cannot join the lobby again. Only the lobby creator may do it, and they
// cannot ban themselves.
func (b *Bot) onBan(ctx context.Context, _ *botClient, update *models.Update) {
	q := update.CallbackQuery
	if q == nil {
		return
	}
	tr := b.tr(ctx, q.From.ID)

	lobbyID, memberID, ok := b.parseTwoIDs(ctx, q, cbBan)
	if !ok {
		return
	}

	lobby, err := b.store.GetLobby(ctx, lobbyID)
	if err != nil {
		b.answer(ctx, q.ID, tr.T(i18n.KeyLobbyGone))
		return
	}
	if lobby.CreatorID != q.From.ID {
		b.answer(ctx, q.ID, tr.T(i18n.KeyAdminOnly))
		return
	}
	if memberID == lobby.CreatorID {
		b.answer(ctx, q.ID, tr.T(i18n.KeyCantBanSelf))
		return
	}

	if _, err := b.store.AddMember(ctx, db.AddMemberParams{
		LobbyID: lobbyID,
		UserID:  memberID,
		Status:  db.MembershipStatusBanned,
	}); err != nil {
		b.log.Error("ban member", zap.Error(err))
		b.answer(ctx, q.ID, tr.T(i18n.KeyErrGeneric))
		return
	}

	b.answer(ctx, q.ID, tr.T(i18n.KeyBannedToast))
	b.editDecision(ctx, q, tr.T(i18n.KeyBannedEdit, htmlEscape(lobby.Name)))
	b.send(ctx, memberID, b.tr(ctx, memberID).T(i18n.KeyBannedMsg, htmlEscape(lobby.Name)), nil)
}

// editDecision replaces the admin's prompt with the outcome, removing buttons.
func (b *Bot) editDecision(ctx context.Context, q *models.CallbackQuery, text string) {
	if q.Message.Message == nil {
		return
	}
	_, err := b.api.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    q.Message.Message.Chat.ID,
		MessageID: q.Message.Message.ID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		b.log.Warn("edit decision message failed", zap.Error(err))
	}
}

// parseSingleID extracts one int64 argument from callback data "action:<id>".
func (b *Bot) parseSingleID(ctx context.Context, q *models.CallbackQuery, action string) (int64, bool) {
	rest := strings.TrimPrefix(q.Data, action+":")
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		b.answer(ctx, q.ID, b.tr(ctx, q.From.ID).T(i18n.KeyMalformed))
		return 0, false
	}
	return id, true
}

// parseTwoIDs extracts two int64 arguments from "action:<a>:<b>".
func (b *Bot) parseTwoIDs(ctx context.Context, q *models.CallbackQuery, action string) (int64, int64, bool) {
	rest := strings.TrimPrefix(q.Data, action+":")
	a, bStr, found := strings.Cut(rest, ":")
	if !found {
		b.answer(ctx, q.ID, b.tr(ctx, q.From.ID).T(i18n.KeyMalformed))
		return 0, 0, false
	}
	first, err1 := strconv.ParseInt(a, 10, 64)
	second, err2 := strconv.ParseInt(bStr, 10, 64)
	if err1 != nil || err2 != nil {
		b.answer(ctx, q.ID, b.tr(ctx, q.From.ID).T(i18n.KeyMalformed))
		return 0, 0, false
	}
	return first, second, true
}

// userFromModel adapts a telegram user into the db.User shape used by
// describeUser (only the display fields matter here).
func userFromModel(u models.User) db.User {
	return db.User{
		ID:        u.ID,
		Username:  nullableText(u.Username),
		FirstName: u.FirstName,
		LastName:  u.LastName,
	}
}

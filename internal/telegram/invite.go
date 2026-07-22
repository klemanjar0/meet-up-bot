package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"meet-up-bot/internal/i18n"
	"meet-up-bot/internal/storage/db"
)

// inviteLink builds the shareable deep link for joining a lobby.
func (b *Bot) inviteLink(lobbyID int64) string {
	return fmt.Sprintf("https://t.me/%s?start=%s%d", b.username, startPayloadJoin, lobbyID)
}

// parseJoinPayload extracts a lobby ID from a "join_<id>" deep-link payload.
func parseJoinPayload(payload string) (int64, bool) {
	rest, ok := strings.CutPrefix(payload, startPayloadJoin)
	if !ok {
		return 0, false
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

// onInvite gives an admin or member a shareable invite link for a lobby.
func (b *Bot) onInvite(ctx context.Context, _ *botClient, update *models.Update) {
	q := update.CallbackQuery
	if q == nil {
		return
	}
	tr := b.tr(ctx, q.From.ID)

	lobbyID, ok := b.parseSingleID(ctx, q, cbInvite)
	if !ok {
		return
	}
	lobby, err := b.store.GetLobby(ctx, lobbyID)
	if err != nil {
		b.answer(ctx, q.ID, tr.T(i18n.KeyLobbyGone))
		return
	}

	// Only people already in the lobby (admin or approved member) can invite.
	isAdmin := lobby.CreatorID == q.From.ID
	status, isMember := b.memberStatus(ctx, lobbyID, q.From.ID)
	if !isAdmin && !(isMember && status == db.MembershipStatusApproved) {
		b.answer(ctx, q.ID, tr.T(i18n.KeyNotAMember))
		return
	}
	if b.username == "" {
		b.answer(ctx, q.ID, tr.T(i18n.KeyErrGeneric))
		return
	}

	b.answer(ctx, q.ID, "")
	b.send(ctx, q.From.ID, tr.T(i18n.KeyInviteText, htmlEscape(lobby.Name), b.inviteLink(lobbyID)), nil)
}

// startJoin runs the join/request flow triggered by tapping an invite deep link
// ("/start join_<id>"). It mirrors the Join button but replies with messages
// rather than callback toasts.
func (b *Bot) startJoin(ctx context.Context, tr *i18n.Translator, msg *models.Message, lobbyID int64) {
	lobby, err := b.store.GetLobby(ctx, lobbyID)
	if errors.Is(err, pgx.ErrNoRows) {
		b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyLobbyGone), nil)
		return
	}
	if err != nil {
		b.log.Error("get lobby on invite join", zap.Error(err))
		b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyErrGeneric), nil)
		return
	}

	// Already related to this lobby: report the current state.
	if existing, err := b.store.GetMember(ctx, db.GetMemberParams{LobbyID: lobbyID, UserID: msg.From.ID}); err == nil {
		switch existing.Status {
		case db.MembershipStatusApproved:
			b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyAlreadyIn), nil)
		case db.MembershipStatusPending:
			b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyStillPending), nil)
		case db.MembershipStatusRejected:
			b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyWasDeclined), nil)
		case db.MembershipStatusBanned:
			b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyBanned), nil)
		}
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		b.log.Error("get member on invite join", zap.Error(err))
		b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyErrGeneric), nil)
		return
	}

	if lobby.Visibility == db.LobbyVisibilityPublic {
		if _, err := b.store.AddMember(ctx, db.AddMemberParams{LobbyID: lobbyID, UserID: msg.From.ID, Status: db.MembershipStatusApproved}); err != nil {
			b.log.Error("add member via invite", zap.Error(err))
			b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyErrGeneric), nil)
			return
		}
		b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyJoinedMsg, htmlEscape(lobby.Name)), nil)
		if lobby.ChatLink != nil && *lobby.ChatLink != "" {
			b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyChatLine, htmlEscape(*lobby.ChatLink)), nil)
		}
		return
	}

	// Private: create a pending request and notify the admin.
	if _, err := b.store.AddMember(ctx, db.AddMemberParams{LobbyID: lobbyID, UserID: msg.From.ID, Status: db.MembershipStatusPending}); err != nil {
		b.log.Error("add pending member via invite", zap.Error(err))
		b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyErrGeneric), nil)
		return
	}
	b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyRequestSentMsg, htmlEscape(lobby.Name)), nil)
	b.notifyRequestToAdmin(ctx, lobby, *msg.From)
}

// notifyRequestToAdmin sends the lobby creator an approve/reject prompt for a
// new join request (in the admin's own locale).
func (b *Bot) notifyRequestToAdmin(ctx context.Context, lobby db.Lobby, from models.User) {
	adminTr := b.tr(ctx, lobby.CreatorID)
	who := describeUser(userFromModel(from))
	markup := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{{
			{Text: adminTr.T(i18n.KeyBtnApprove), CallbackData: fmt.Sprintf("%s:%d:%d", cbApprove, lobby.ID, from.ID)},
			{Text: adminTr.T(i18n.KeyBtnReject), CallbackData: fmt.Sprintf("%s:%d:%d", cbReject, lobby.ID, from.ID)},
		}},
	}
	b.send(ctx, lobby.CreatorID, adminTr.T(i18n.KeyAdminNewRequest, who, htmlEscape(lobby.Name)), markup)
}

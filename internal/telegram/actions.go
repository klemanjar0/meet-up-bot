package telegram

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"

	"meet-up-bot/internal/i18n"
	"meet-up-bot/internal/storage/db"
)

// onDeleteLobby asks the admin to confirm deleting a lobby.
func (b *Bot) onDeleteLobby(ctx context.Context, _ *botClient, update *models.Update) {
	q := update.CallbackQuery
	if q == nil {
		return
	}
	tr := b.tr(ctx, q.From.ID)

	lobbyID, ok := b.parseSingleID(ctx, q, cbDelete)
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

	markup := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{{
			{Text: tr.T(i18n.KeyBtnConfirmDelete), CallbackData: fmt.Sprintf("%s:%d", cbDeleteYes, lobby.ID)},
			{Text: tr.T(i18n.KeyBtnCancel), CallbackData: cbDismiss + ":0"},
		}},
	}
	b.send(ctx, q.From.ID, tr.T(i18n.KeyDeleteConfirm, htmlEscape(lobby.Name)), markup)
}

// onDeleteConfirm deletes the lobby and notifies its approved members.
func (b *Bot) onDeleteConfirm(ctx context.Context, _ *botClient, update *models.Update) {
	q := update.CallbackQuery
	if q == nil {
		return
	}
	tr := b.tr(ctx, q.From.ID)

	lobbyID, ok := b.parseSingleID(ctx, q, cbDeleteYes)
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

	// Capture members before the row (and its memberships) are gone.
	members, err := b.store.ListApprovedMembers(ctx, lobbyID)
	if err != nil {
		b.log.Error("list members before delete", zap.Error(err))
	}

	if err := b.store.DeleteLobby(ctx, db.DeleteLobbyParams{ID: lobbyID, CreatorID: q.From.ID}); err != nil {
		b.log.Error("delete lobby", zap.Error(err))
		b.answer(ctx, q.ID, tr.T(i18n.KeyErrGeneric))
		return
	}

	b.answer(ctx, q.ID, tr.T(i18n.KeyDeletedToast))
	b.editDecision(ctx, q, tr.T(i18n.KeyDeletedEdit, htmlEscape(lobby.Name)))

	for _, m := range members {
		if m.UserID == lobby.CreatorID {
			continue
		}
		mtr := b.tr(ctx, m.UserID)
		b.send(ctx, m.UserID, mtr.T(i18n.KeyDeletedNotify, htmlEscape(lobby.Name)), nil)
	}
}

// onLeaveLobby removes the caller's membership from a lobby.
func (b *Bot) onLeaveLobby(ctx context.Context, _ *botClient, update *models.Update) {
	q := update.CallbackQuery
	if q == nil {
		return
	}
	tr := b.tr(ctx, q.From.ID)

	lobbyID, ok := b.parseSingleID(ctx, q, cbLeave)
	if !ok {
		return
	}
	lobby, err := b.store.GetLobby(ctx, lobbyID)
	if err != nil {
		b.answer(ctx, q.ID, tr.T(i18n.KeyLobbyGone))
		return
	}
	// The creator can't leave their own lobby — they delete it instead.
	if lobby.CreatorID == q.From.ID {
		b.answer(ctx, q.ID, tr.T(i18n.KeyAdminOnly))
		return
	}

	if err := b.store.RemoveMember(ctx, db.RemoveMemberParams{LobbyID: lobbyID, UserID: q.From.ID}); err != nil {
		b.log.Error("leave lobby", zap.Error(err))
		b.answer(ctx, q.ID, tr.T(i18n.KeyErrGeneric))
		return
	}
	b.answer(ctx, q.ID, tr.T(i18n.KeyLeftToast))
	b.editDecision(ctx, q, tr.T(i18n.KeyLeftEdit, htmlEscape(lobby.Name)))
}

// onDismiss acknowledges a cancelled action and clears the prompt's buttons.
func (b *Bot) onDismiss(ctx context.Context, _ *botClient, update *models.Update) {
	q := update.CallbackQuery
	if q == nil {
		return
	}
	tr := b.tr(ctx, q.From.ID)
	b.answer(ctx, q.ID, "")
	b.editDecision(ctx, q, tr.T(i18n.KeyCancelled))
}

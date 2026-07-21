package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"

	"meet-up-bot/internal/i18n"
	"meet-up-bot/internal/storage/db"
)

// onEdit shows the field menu for a lobby the caller owns.
func (b *Bot) onEdit(ctx context.Context, _ *botClient, update *models.Update) {
	q := update.CallbackQuery
	if q == nil {
		return
	}
	tr := b.tr(ctx, q.From.ID)

	lobbyID, ok := b.parseSingleID(ctx, q, cbEdit)
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
	b.send(ctx, q.From.ID, tr.T(i18n.KeyEditMenu, htmlEscape(lobby.Name)), editMenuMarkup(tr, lobby))
}

func editMenuMarkup(tr *i18n.Translator, lobby db.Lobby) *models.InlineKeyboardMarkup {
	visBtn := tr.T(i18n.KeyBtnMakePrivate)
	if lobby.Visibility == db.LobbyVisibilityPrivate {
		visBtn = tr.T(i18n.KeyBtnMakePublic)
	}
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: tr.T(i18n.KeyBtnEditName), CallbackData: editFieldData(lobby.ID, editFieldName)},
				{Text: tr.T(i18n.KeyBtnEditTime), CallbackData: editFieldData(lobby.ID, editFieldTime)},
			},
			{
				{Text: tr.T(i18n.KeyBtnEditCountry), CallbackData: editFieldData(lobby.ID, editFieldCountry)},
				{Text: tr.T(i18n.KeyBtnEditCity), CallbackData: editFieldData(lobby.ID, editFieldCity)},
			},
			{
				{Text: tr.T(i18n.KeyBtnEditAddress), CallbackData: editFieldData(lobby.ID, editFieldAddress)},
				{Text: tr.T(i18n.KeyBtnEditLink), CallbackData: editFieldData(lobby.ID, editFieldLink)},
			},
			{
				{Text: visBtn, CallbackData: fmt.Sprintf("%s:%d", cbEditVis, lobby.ID)},
			},
		},
	}
}

func editFieldData(lobbyID int64, f editField) string {
	return fmt.Sprintf("%s:%d:%s", cbEditField, lobbyID, f)
}

// onEditField begins collecting a new value for one text field of a lobby.
func (b *Bot) onEditField(ctx context.Context, _ *botClient, update *models.Update) {
	q := update.CallbackQuery
	if q == nil {
		return
	}
	tr := b.tr(ctx, q.From.ID)

	rest := strings.TrimPrefix(q.Data, cbEditField+":")
	idStr, fieldStr, found := strings.Cut(rest, ":")
	if !found {
		b.answer(ctx, q.ID, tr.T(i18n.KeyMalformed))
		return
	}
	lobbyID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		b.answer(ctx, q.ID, tr.T(i18n.KeyMalformed))
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

	field := editField(fieldStr)
	var prompt string
	switch field {
	case editFieldName:
		prompt = tr.T(i18n.KeyEditAskName)
	case editFieldCountry:
		prompt = tr.T(i18n.KeyEditAskCountry)
	case editFieldCity:
		prompt = tr.T(i18n.KeyEditAskCity)
	case editFieldAddress:
		prompt = tr.T(i18n.KeyEditAskAddress)
	case editFieldTime:
		prompt = tr.T(i18n.KeyEditAskTime, displayTime, b.userTimezone(ctx, q.From.ID))
	case editFieldLink:
		prompt = tr.T(i18n.KeyEditAskLink)
	default:
		b.answer(ctx, q.ID, tr.T(i18n.KeyMalformed))
		return
	}

	b.answer(ctx, q.ID, "")
	b.sessions.startEdit(q.From.ID, lobbyID, field)
	b.send(ctx, q.From.ID, prompt, nil)
}

// onEditVisibility flips a lobby between public and private.
func (b *Bot) onEditVisibility(ctx context.Context, _ *botClient, update *models.Update) {
	q := update.CallbackQuery
	if q == nil {
		return
	}
	tr, loc := b.viewer(ctx, q.From.ID)

	lobbyID, ok := b.parseSingleID(ctx, q, cbEditVis)
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

	if lobby.Visibility == db.LobbyVisibilityPrivate {
		lobby.Visibility = db.LobbyVisibilityPublic
	} else {
		lobby.Visibility = db.LobbyVisibilityPrivate
	}
	b.answer(ctx, q.ID, "")
	b.applyLobbyUpdate(ctx, tr, loc, q.From.ID, lobby)
}

// handleEditInput applies a free-text answer to the field being edited.
func (b *Bot) handleEditInput(ctx context.Context, tr *i18n.Translator, msg *models.Message, s *session) {
	text := strings.TrimSpace(msg.Text)

	lobby, err := b.store.GetLobby(ctx, s.editLobbyID)
	if err != nil {
		b.sessions.clear(msg.From.ID)
		b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyLobbyGone), nil)
		return
	}
	if lobby.CreatorID != msg.From.ID {
		b.sessions.clear(msg.From.ID)
		b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyAdminOnly), nil)
		return
	}

	switch s.editField {
	case editFieldName:
		if text == "" {
			b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyNameEmpty), nil)
			return
		}
		lobby.Name = text
	case editFieldCountry:
		if text == "" {
			b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyCountryEmpty), nil)
			return
		}
		lobby.Country = text
	case editFieldCity:
		if text == "" {
			b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyCityEmpty), nil)
			return
		}
		lobby.City = text
	case editFieldAddress:
		if text == "-" {
			lobby.Address = nil
		} else {
			lobby.Address = nullableText(text)
		}
	case editFieldTime:
		loc := loadLocation(b.userTimezone(ctx, msg.From.ID))
		t, err := parseEventTimeIn(text, loc)
		if err != nil {
			b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyTimeBad, displayTime), nil)
			return
		}
		lobby.EventTime = toTimestamptz(t)
	case editFieldLink:
		if text == "-" {
			lobby.ChatLink = nil
		} else {
			lobby.ChatLink = nullableText(text)
		}
	default:
		b.sessions.clear(msg.From.ID)
		return
	}

	b.sessions.clear(msg.From.ID)
	_, loc := b.viewer(ctx, msg.From.ID)
	b.applyLobbyUpdate(ctx, tr, loc, msg.Chat.ID, lobby)
}

// applyLobbyUpdate persists the modified lobby, confirms to the admin, and
// notifies every approved participant of the change.
func (b *Bot) applyLobbyUpdate(ctx context.Context, tr *i18n.Translator, loc *time.Location, chatID int64, lobby db.Lobby) {
	saved, err := b.store.UpdateLobby(ctx, db.UpdateLobbyParams{
		ID:         lobby.ID,
		Name:       lobby.Name,
		Country:    lobby.Country,
		City:       lobby.City,
		Address:    lobby.Address,
		EventTime:  lobby.EventTime,
		ChatLink:   lobby.ChatLink,
		Visibility: lobby.Visibility,
		CreatorID:  lobby.CreatorID,
	})
	if err != nil {
		b.log.Error("update lobby", zap.Error(err))
		b.send(ctx, chatID, tr.T(i18n.KeyErrGeneric), nil)
		return
	}

	count, _ := b.store.CountApprovedMembers(ctx, saved.ID)
	b.send(ctx, chatID, tr.T(i18n.KeyEditSaved, htmlEscape(saved.Name))+"\n\n"+formatLobby(tr, loc, saved, count, true), nil)
	b.notifyLobbyUpdated(ctx, saved)
}

// notifyLobbyUpdated messages each approved member (except the admin) that the
// lobby changed, in their own locale and timezone, revealing the chat link.
func (b *Bot) notifyLobbyUpdated(ctx context.Context, lobby db.Lobby) {
	members, err := b.store.ListApprovedMembers(ctx, lobby.ID)
	if err != nil {
		b.log.Error("list approved members for notify", zap.Error(err))
		return
	}
	count := int64(len(members))
	for _, m := range members {
		if m.UserID == lobby.CreatorID {
			continue
		}
		tr, loc := b.viewer(ctx, m.UserID)
		text := tr.T(i18n.KeyEditNotify, htmlEscape(lobby.Name)) + "\n\n" + formatLobby(tr, loc, lobby, count, true)
		b.send(ctx, m.UserID, text, nil)
	}
}

package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"meet-up-bot/internal/i18n"
	"meet-up-bot/internal/storage/db"
)

// lobbyPageSize is how many lobbies are shown per page in /lobbies.
const lobbyPageSize = 10

// sendLobbiesPage renders one page of upcoming lobbies for a user, applying
// their city and time-window preferences, soonest first.
func (b *Bot) sendLobbiesPage(ctx context.Context, chatID, userID int64, page int) {
	if page < 0 {
		page = 0
	}
	tr := i18n.For(i18n.En)
	loc := time.UTC
	city, filter := "", filterAll
	if u, err := b.store.GetUser(ctx, userID); err == nil {
		tr = i18n.For(i18n.Parse(u.Locale))
		loc = loadLocation(u.Timezone)
		city = u.City
		filter = u.TimeFilter
	}

	// Fetch one extra row to detect whether a next page exists.
	rows, err := b.store.ListLobbiesFiltered(ctx, db.ListLobbiesFilteredParams{
		City:  city,
		Until: filterUntil(filter),
		Off:   int32(page * lobbyPageSize),
		Lim:   int32(lobbyPageSize + 1),
	})
	if err != nil {
		b.log.Error("list lobbies filtered", zap.Error(err))
		b.send(ctx, chatID, tr.T(i18n.KeyErrLoadLobbies), nil)
		return
	}

	hasNext := len(rows) > lobbyPageSize
	if hasNext {
		rows = rows[:lobbyPageSize]
	}

	if len(rows) == 0 {
		if page == 0 {
			b.send(ctx, chatID, tr.T(i18n.KeyNoLobbies), nil)
		}
		return
	}

	for _, l := range rows {
		text, markup := b.renderLobbyListItem(ctx, tr, loc, l, userID)
		b.send(ctx, chatID, text, markup)
	}

	// Navigation footer: active filters, page number, and prev/next buttons.
	note := tr.T(i18n.KeyLobbiesFilterNote, filterCityLabel(tr, city), filterLabel(tr, filter))
	footer := note + "\n" + tr.T(i18n.KeyPageLabel, page+1)
	b.send(ctx, chatID, footer, pageNavMarkup(tr, page, hasNext))
}

// renderLobbyListItem builds the message and (optional) join button for one
// lobby, reflecting the user's existing relationship with it.
func (b *Bot) renderLobbyListItem(ctx context.Context, tr *i18n.Translator, loc *time.Location, l db.Lobby, userID int64) (string, models.ReplyMarkup) {
	count, _ := b.store.CountApprovedMembers(ctx, l.ID)
	private := l.Visibility == db.LobbyVisibilityPrivate

	// The chat link is only revealed to people already in the lobby: approved
	// members or the admin (creator). Everyone else — including browsers of
	// public lobbies — must join first.
	showLink := l.CreatorID == userID

	var markup models.ReplyMarkup
	statusLine := ""
	if status, isMember := b.memberStatus(ctx, l.ID, userID); isMember {
		// Already related to this lobby: show status instead of a join button.
		switch status {
		case db.MembershipStatusApproved:
			statusLine = tr.T(i18n.KeyStatusJoined)
			showLink = true
		case db.MembershipStatusPending:
			statusLine = tr.T(i18n.KeyStatusPending)
		case db.MembershipStatusRejected:
			statusLine = tr.T(i18n.KeyWasDeclined)
		case db.MembershipStatusBanned:
			statusLine = tr.T(i18n.KeyBanned)
		}
	} else {
		joinLabel := tr.T(i18n.KeyBtnJoin)
		if private {
			joinLabel = tr.T(i18n.KeyBtnRequestJoin)
		}
		markup = &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{{
				{Text: joinLabel, CallbackData: fmt.Sprintf("%s:%d", cbJoin, l.ID)},
			}},
		}
	}

	text := formatLobby(tr, loc, l, count, showLink)
	if statusLine != "" {
		text += "\n" + statusLine
	}
	return text, markup
}

// onLobbyPage handles the prev/next pagination buttons.
func (b *Bot) onLobbyPage(ctx context.Context, _ *botClient, update *models.Update) {
	q := update.CallbackQuery
	if q == nil {
		return
	}
	b.answer(ctx, q.ID, "")
	page, ok := b.parseSingleID(ctx, q, cbLobbyPage)
	if !ok {
		return
	}
	b.sendLobbiesPage(ctx, q.From.ID, q.From.ID, int(page))
}

// pageNavMarkup returns the prev/next navigation row, or a true nil interface
// when there is nothing to navigate to. The return type is the ReplyMarkup
// interface (not the concrete pointer) so a nil result stays a nil interface —
// returning a typed nil pointer here would serialize to an invalid "null"
// reply markup and Telegram rejects the send.
func pageNavMarkup(tr *i18n.Translator, page int, hasNext bool) models.ReplyMarkup {
	var row []models.InlineKeyboardButton
	if page > 0 {
		row = append(row, models.InlineKeyboardButton{
			Text: tr.T(i18n.KeyBtnPrev), CallbackData: fmt.Sprintf("%s:%d", cbLobbyPage, page-1),
		})
	}
	if hasNext {
		row = append(row, models.InlineKeyboardButton{
			Text: tr.T(i18n.KeyBtnNext), CallbackData: fmt.Sprintf("%s:%d", cbLobbyPage, page+1),
		})
	}
	if len(row) == 0 {
		return nil
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{row}}
}

// filterUntil maps a stored time-filter value to an upper time bound, or a NULL
// (skip filter) for "all".
func filterUntil(filter string) pgtype.Timestamptz {
	var d time.Duration
	switch filter {
	case filterDay:
		d = 24 * time.Hour
	case filterWeek:
		d = 7 * 24 * time.Hour
	case filterMonth:
		d = 30 * 24 * time.Hour
	default:
		return pgtype.Timestamptz{} // NULL — no upper bound
	}
	return pgtype.Timestamptz{Time: time.Now().Add(d), Valid: true}
}

// filterLabel returns the localized name of a time-filter value.
func filterLabel(tr *i18n.Translator, filter string) string {
	switch filter {
	case filterDay:
		return tr.T(i18n.KeyBtnFilterDay)
	case filterWeek:
		return tr.T(i18n.KeyBtnFilterWeek)
	case filterMonth:
		return tr.T(i18n.KeyBtnFilterMonth)
	default:
		return tr.T(i18n.KeyBtnFilterAll)
	}
}

// filterCityLabel returns the city or a localized "not set" placeholder.
func filterCityLabel(tr *i18n.Translator, city string) string {
	if city == "" {
		return tr.T(i18n.KeyNotSet)
	}
	return city
}

package telegram

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"meet-up-bot/internal/i18n"
	"meet-up-bot/internal/storage/db"
)

// Callback action names. Callback data is encoded as "<action>:<args...>".
const (
	cbVisibility = "vis"
	cbJoin       = "join"
	cbApprove    = "approve"
	cbReject     = "reject"
	cbMembers    = "members"
	cbRemove     = "remove"
	cbBan        = "ban"
	cbEdit       = "edit"
	cbEditField  = "editf"
	cbEditVis    = "editvis"
	cbLocale     = "locale"
	cbDetails    = "details"
	cbSettings   = "setmenu"
	cbTimeFilter = "tf"
	cbLobbyPage  = "lobbypage"
	cbDelete     = "del"
	cbDeleteYes  = "delyes"
	cbLeave      = "leave"
	cbDismiss    = "dismiss"
)

// timeFilter values stored in users.time_filter.
const (
	filterDay   = "day"
	filterWeek  = "week"
	filterMonth = "month"
	filterAll   = "all"
)

// displayTime is the layout users type; displayTimeTZ additionally shows the
// timezone abbreviation when rendering a stored time back to a viewer.
const (
	displayTime   = "2006-01-02 15:04"
	displayTimeTZ = "2006-01-02 15:04 MST"
)

// timeLayouts are accepted when parsing user-entered event times.
var timeLayouts = []string{
	"2006-01-02 15:04",
	"2006-01-02T15:04",
	"02.01.2006 15:04",
	"2006-01-02 15:04:05",
}

// validTimezone reports whether tz is a non-empty, loadable IANA timezone name.
func validTimezone(tz string) bool {
	if strings.TrimSpace(tz) == "" {
		return false
	}
	_, err := time.LoadLocation(tz)
	return err == nil
}

// loadLocation resolves an IANA timezone name, falling back to UTC.
func loadLocation(tz string) *time.Location {
	if tz == "" {
		return time.UTC
	}
	if loc, err := time.LoadLocation(tz); err == nil {
		return loc
	}
	return time.UTC
}

// parseEventTimeIn parses a user-entered time, interpreting the wall-clock value
// as being in loc (so "19:00" means 19:00 in the user's own timezone).
func parseEventTimeIn(s string, loc *time.Location) (time.Time, error) {
	for _, layout := range timeLayouts {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("could not parse time %q", s)
}

// toTimestamptz converts a time.Time into the pgtype used by sqlc.
func toTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// nullableText converts a possibly-empty string into a *string for nullable
// columns (nil when empty).
func nullableText(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// htmlEscape escapes a string for inclusion in an HTML-parse-mode message.
func htmlEscape(s string) string { return html.EscapeString(s) }

// describeUser produces a human-friendly label for a user: "@username" when
// available, otherwise their name.
func describeUser(u db.User) string {
	if u.Username != nil && *u.Username != "" {
		return "@" + *u.Username
	}
	name := u.FirstName
	if u.LastName != "" {
		name += " " + u.LastName
	}
	if name == "" {
		return fmt.Sprintf("User %d", u.ID)
	}
	return htmlEscape(name)
}

// formatLobby renders a lobby as an HTML message body in the translator's
// locale, with the event time shown in the viewer's timezone (loc). showLink
// controls whether the Telegram chat link is revealed: for a private lobby it
// must stay hidden in public listings until the viewer has been approved.
func formatLobby(tr *i18n.Translator, loc *time.Location, l db.Lobby, approvedCount int64, showLink bool) string {
	visibility := tr.T(i18n.KeyLobbyPublic)
	if l.Visibility == db.LobbyVisibilityPrivate {
		visibility = tr.T(i18n.KeyLobbyPrivate)
	}

	when := l.EventTime.Time.In(loc).Format(displayTimeTZ)
	s := fmt.Sprintf("<b>%s</b>", htmlEscape(l.Name))
	if l.Description != nil && *l.Description != "" {
		s += "\n<i>" + htmlEscape(*l.Description) + "</i>"
	}
	s += "\n🕒 " + htmlEscape(when)

	if place := locationLine(l); place != "" {
		s += "\n📍 " + htmlEscape(place)
	}
	if l.Address != nil && *l.Address != "" {
		s += "\n🏠 " + htmlEscape(*l.Address)
	}

	s += "\n" + tr.T(i18n.KeyJoinedCount, approvedCount)
	s += "\n" + visibility

	if showLink && l.ChatLink != nil && *l.ChatLink != "" {
		s += fmt.Sprintf("\n💬 %s", htmlEscape(*l.ChatLink))
	}
	return s
}

// locationLine joins the non-empty city and country into "City, Country".
func locationLine(l db.Lobby) string {
	switch {
	case l.City != "" && l.Country != "":
		return l.City + ", " + l.Country
	case l.City != "":
		return l.City
	default:
		return l.Country
	}
}

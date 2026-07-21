package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"

	"meet-up-bot/internal/i18n"
	"meet-up-bot/internal/storage/db"
)

// onSettings shows the settings menu with the user's current preferences.
func (b *Bot) onSettings(ctx context.Context, _ *botClient, update *models.Update) {
	msg := update.Message
	if msg == nil || msg.From == nil {
		return
	}
	user, err := b.ensureUser(ctx, msg.From)
	if err != nil {
		b.log.Error("ensure user", zap.Error(err))
	}
	tr := i18n.For(i18n.Parse(user.Locale))
	b.send(ctx, msg.Chat.ID, settingsSummary(tr, user), settingsMenuMarkup(tr))
}

// settingsMenuMarkup builds the four setting buttons.
func settingsMenuMarkup(tr *i18n.Translator) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: tr.T(i18n.KeyBtnSetLang), CallbackData: cbSettings + ":lang"},
				{Text: tr.T(i18n.KeyBtnSetTimezone), CallbackData: cbSettings + ":tz"},
			},
			{
				{Text: tr.T(i18n.KeyBtnSetCity), CallbackData: cbSettings + ":city"},
				{Text: tr.T(i18n.KeyBtnSetTimeFilter), CallbackData: cbSettings + ":filter"},
			},
		},
	}
}

// settingsSummary renders the settings menu body with current values.
func settingsSummary(tr *i18n.Translator, u db.User) string {
	lang := "English"
	if i18n.Parse(u.Locale) == i18n.Ru {
		lang = "Русский"
	}
	city := u.City
	if city == "" {
		city = tr.T(i18n.KeyNotSet)
	}
	return tr.T(i18n.KeySettingsMenu, lang, u.Timezone, city, filterLabel(tr, u.TimeFilter))
}

// onSettingsMenu routes a top-level settings button to its sub-flow.
func (b *Bot) onSettingsMenu(ctx context.Context, _ *botClient, update *models.Update) {
	q := update.CallbackQuery
	if q == nil {
		return
	}
	tr := b.tr(ctx, q.From.ID)
	b.answer(ctx, q.ID, "")

	_, which, _ := strings.Cut(q.Data, ":")
	switch which {
	case "lang":
		markup := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{{
				{Text: tr.T(i18n.KeyBtnEnglish), CallbackData: fmt.Sprintf("%s:%s", cbLocale, i18n.En)},
				{Text: tr.T(i18n.KeyBtnRussian), CallbackData: fmt.Sprintf("%s:%s", cbLocale, i18n.Ru)},
			}},
		}
		b.send(ctx, q.From.ID, tr.T(i18n.KeySettingsPrompt), markup)
	case "tz":
		b.sessions.startSettings(q.From.ID, settingsFieldTimezone)
		b.send(ctx, q.From.ID, tr.T(i18n.KeyAskTimezone), nil)
	case "city":
		b.sessions.startSettings(q.From.ID, settingsFieldCity)
		b.send(ctx, q.From.ID, tr.T(i18n.KeyAskCitySetting), nil)
	case "filter":
		b.send(ctx, q.From.ID, tr.T(i18n.KeyTimeFilterPrompt), timeFilterMarkup(tr))
	}
}

func timeFilterMarkup(tr *i18n.Translator) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: tr.T(i18n.KeyBtnFilterDay), CallbackData: cbTimeFilter + ":" + filterDay},
				{Text: tr.T(i18n.KeyBtnFilterWeek), CallbackData: cbTimeFilter + ":" + filterWeek},
			},
			{
				{Text: tr.T(i18n.KeyBtnFilterMonth), CallbackData: cbTimeFilter + ":" + filterMonth},
				{Text: tr.T(i18n.KeyBtnFilterAll), CallbackData: cbTimeFilter + ":" + filterAll},
			},
		},
	}
}

// onLocale stores the user's chosen language and confirms in that language.
func (b *Bot) onLocale(ctx context.Context, _ *botClient, update *models.Update) {
	q := update.CallbackQuery
	if q == nil {
		return
	}
	if _, err := b.ensureUser(ctx, &q.From); err != nil {
		b.log.Error("ensure user", zap.Error(err))
	}

	_, value, _ := strings.Cut(q.Data, ":")
	loc := i18n.Parse(value)
	if err := b.store.SetUserLocale(ctx, db.SetUserLocaleParams{ID: q.From.ID, Locale: string(loc)}); err != nil {
		b.log.Error("set locale", zap.Error(err))
		b.answer(ctx, q.ID, "")
		return
	}
	b.answer(ctx, q.ID, "")
	b.send(ctx, q.From.ID, i18n.For(loc).T(i18n.KeyLocaleSet), nil)
}

// onTimeFilter stores the user's lobby time-window preference.
func (b *Bot) onTimeFilter(ctx context.Context, _ *botClient, update *models.Update) {
	q := update.CallbackQuery
	if q == nil {
		return
	}
	tr := b.tr(ctx, q.From.ID)
	b.answer(ctx, q.ID, "")

	_, value, _ := strings.Cut(q.Data, ":")
	switch value {
	case filterDay, filterWeek, filterMonth, filterAll:
	default:
		return
	}
	if err := b.store.SetUserTimeFilter(ctx, db.SetUserTimeFilterParams{ID: q.From.ID, TimeFilter: value}); err != nil {
		b.log.Error("set time filter", zap.Error(err))
		return
	}
	b.send(ctx, q.From.ID, tr.T(i18n.KeyTimeFilterSet), nil)
}

// handleSettingsInput applies a free-text answer for the timezone or city
// setting.
func (b *Bot) handleSettingsInput(ctx context.Context, tr *i18n.Translator, msg *models.Message, s *session) {
	text := strings.TrimSpace(msg.Text)

	switch s.settingsField {
	case settingsFieldTimezone:
		if !validTimezone(text) {
			b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyBadTimezone), nil)
			return
		}
		if err := b.store.SetUserTimezone(ctx, db.SetUserTimezoneParams{ID: msg.From.ID, Timezone: text}); err != nil {
			b.log.Error("set timezone", zap.Error(err))
			b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyErrGeneric), nil)
			return
		}
		b.sessions.clear(msg.From.ID)
		b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyTimezoneSet, htmlEscape(text)), nil)

	case settingsFieldCity:
		b.sessions.clear(msg.From.ID)
		if text == "-" {
			text = ""
		}
		if err := b.store.SetUserCity(ctx, db.SetUserCityParams{ID: msg.From.ID, City: text}); err != nil {
			b.log.Error("set city", zap.Error(err))
			b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyErrGeneric), nil)
			return
		}
		if text == "" {
			b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyCityCleared), nil)
		} else {
			b.send(ctx, msg.Chat.ID, tr.T(i18n.KeyCitySet, htmlEscape(text)), nil)
		}

	default:
		b.sessions.clear(msg.From.ID)
	}
}

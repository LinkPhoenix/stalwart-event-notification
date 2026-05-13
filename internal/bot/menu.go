package bot

import telegrammodels "github.com/go-telegram/bot/models"

const (
	menuEvents         = "Events"
	menuList           = "My subscriptions"
	menuSubscribe      = "Subscribe"
	menuUnsubscribe    = "Unsubscribe"
	menuSubscribeAll   = "Subscribe all"
	menuUnsubscribeAll = "Unsubscribe all"
	menuStatus         = "Status"
	menuPrefs          = "Preferences"
	menuHelp           = "Help"

	callbackSubscribePrefix   = "sub:"
	callbackUnsubscribePrefix = "unsub:"
	callbackPrefsPrefix       = "prefs:"
)

func mainMenuKeyboard() *telegrammodels.ReplyKeyboardMarkup {
	return &telegrammodels.ReplyKeyboardMarkup{
		Keyboard: [][]telegrammodels.KeyboardButton{
			{{Text: menuEvents}, {Text: menuList}},
			{{Text: menuSubscribe}, {Text: menuUnsubscribe}},
			{{Text: menuSubscribeAll}, {Text: menuUnsubscribeAll}},
			{{Text: menuStatus}, {Text: menuPrefs}, {Text: menuHelp}},
		},
		ResizeKeyboard: true,
		IsPersistent:   true,
	}
}

func eventInlineKeyboard(prefix string, eventTypes []string) *telegrammodels.InlineKeyboardMarkup {
	rows := make([][]telegrammodels.InlineKeyboardButton, 0, len(eventTypes))
	for _, eventType := range eventTypes {
		rows = append(rows, []telegrammodels.InlineKeyboardButton{{
			Text:         eventType,
			CallbackData: prefix + eventType,
		}})
	}
	return &telegrammodels.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func prefsInlineKeyboard(locale string, translator Translator) *telegrammodels.InlineKeyboardMarkup {
	return &telegrammodels.InlineKeyboardMarkup{InlineKeyboard: [][]telegrammodels.InlineKeyboardButton{
		{{Text: translator.T(locale, "prefs.language", nil), CallbackData: callbackPrefsPrefix + "lang"}},
		{{Text: translator.T(locale, "prefs.timezone", nil), CallbackData: callbackPrefsPrefix + "timezone"}},
		{{Text: translator.T(locale, "prefs.short", nil), CallbackData: callbackPrefsPrefix + "short"}},
	}}
}

func languageInlineKeyboard() *telegrammodels.InlineKeyboardMarkup {
	return &telegrammodels.InlineKeyboardMarkup{InlineKeyboard: [][]telegrammodels.InlineKeyboardButton{
		{{Text: "English", CallbackData: callbackPrefsPrefix + "lang:en"}, {Text: "Français", CallbackData: callbackPrefsPrefix + "lang:fr"}},
		{{Text: "Deutsch", CallbackData: callbackPrefsPrefix + "lang:de"}, {Text: "Español", CallbackData: callbackPrefsPrefix + "lang:es"}, {Text: "Italiano", CallbackData: callbackPrefsPrefix + "lang:it"}},
	}}
}

func timezoneInlineKeyboard() *telegrammodels.InlineKeyboardMarkup {
	return &telegrammodels.InlineKeyboardMarkup{InlineKeyboard: [][]telegrammodels.InlineKeyboardButton{
		{{Text: "UTC", CallbackData: callbackPrefsPrefix + "tz:UTC"}, {Text: "Europe/Paris", CallbackData: callbackPrefsPrefix + "tz:Europe/Paris"}},
		{{Text: "America/New_York", CallbackData: callbackPrefsPrefix + "tz:America/New_York"}, {Text: "Asia/Tokyo", CallbackData: callbackPrefsPrefix + "tz:Asia/Tokyo"}},
	}}
}

func shortInlineKeyboard(locale string, current bool, translator Translator) *telegrammodels.InlineKeyboardMarkup {
	on := translator.T(locale, "prefs.short_on", nil)
	off := translator.T(locale, "prefs.short_off", nil)
	if current {
		on = "✓ " + on
	} else {
		off = "✓ " + off
	}
	return &telegrammodels.InlineKeyboardMarkup{InlineKeyboard: [][]telegrammodels.InlineKeyboardButton{
		{{Text: on, CallbackData: callbackPrefsPrefix + "short:on"}, {Text: off, CallbackData: callbackPrefsPrefix + "short:off"}},
	}}
}

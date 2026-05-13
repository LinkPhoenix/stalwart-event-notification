package bot

import telegrammodels "github.com/go-telegram/bot/models"

const (
	menuEvents         = "📋 Events"
	menuList           = "📌 My subscriptions"
	menuSubscribe      = "➕ Subscribe"
	menuUnsubscribe    = "➖ Unsubscribe"
	menuSubscribeAll   = "➕ Subscribe all"
	menuUnsubscribeAll = "➖ Unsubscribe all"
	menuStatus         = "📊 Status"
	menuPrefs          = "⚙️ Preferences"
	menuHelp           = "❓ Help"

	callbackSubscribePrefix   = "sub:"
	callbackUnsubscribePrefix = "unsub:"
	callbackPrefsPrefix       = "prefs:"

	buttonStylePrimary = "primary"
	buttonStyleSuccess = "success"
	buttonStyleDanger  = "danger"
)

type supportedLocale struct {
	Code string
	Flag string
}

var supportedLocales = []supportedLocale{
	{Code: "de", Flag: "🇩🇪"},
	{Code: "en", Flag: "🇬🇧"},
	{Code: "es", Flag: "🇪🇸"},
	{Code: "fr", Flag: "🇫🇷"},
	{Code: "it", Flag: "🇮🇹"},
	{Code: "pt", Flag: "🇵🇹"},
	{Code: "ru", Flag: "🇷🇺"},
	{Code: "uk", Flag: "🇺🇦"},
}

var commonTimezones = []string{
	"UTC",
	"Europe/Paris",
	"Europe/London",
	"Europe/Berlin",
	"Europe/Lisbon",
	"Europe/Kyiv",
	"Europe/Moscow",
	"America/New_York",
	"America/Los_Angeles",
	"America/Sao_Paulo",
	"Africa/Casablanca",
	"Asia/Dubai",
	"Asia/Tokyo",
	"Asia/Shanghai",
	"Australia/Sydney",
}

func mainMenuKeyboard() *telegrammodels.ReplyKeyboardMarkup {
	return &telegrammodels.ReplyKeyboardMarkup{
		Keyboard: [][]telegrammodels.KeyboardButton{
			{{Text: menuEvents}, {Text: menuList}},
			{{Text: menuSubscribe, Style: buttonStyleSuccess}, {Text: menuUnsubscribe, Style: buttonStyleDanger}},
			{{Text: menuSubscribeAll, Style: buttonStyleSuccess}, {Text: menuUnsubscribeAll, Style: buttonStyleDanger}},
			{{Text: menuStatus}, {Text: menuPrefs, Style: buttonStylePrimary}, {Text: menuHelp}},
		},
		ResizeKeyboard: true,
		IsPersistent:   true,
	}
}

func eventInlineKeyboard(prefix string, eventTypes []string, locale string, translator Translator) *telegrammodels.InlineKeyboardMarkup {
	rows := make([][]telegrammodels.InlineKeyboardButton, 0, len(eventTypes)+1)
	for _, eventType := range eventTypes {
		rows = append(rows, []telegrammodels.InlineKeyboardButton{{
			Text:         eventType,
			CallbackData: prefix + eventType,
		}})
	}
	rows = append(rows, []telegrammodels.InlineKeyboardButton{primaryExitButton(locale, translator)})
	return &telegrammodels.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func prefsInlineKeyboard(locale string, translator Translator) *telegrammodels.InlineKeyboardMarkup {
	return &telegrammodels.InlineKeyboardMarkup{InlineKeyboard: [][]telegrammodels.InlineKeyboardButton{
		{{Text: "🌐 " + translator.T(locale, "prefs.language", nil), CallbackData: callbackPrefsPrefix + "lang"}},
		{{Text: "🕒 " + translator.T(locale, "prefs.timezone", nil), CallbackData: callbackPrefsPrefix + "timezone"}},
		{{Text: "🔔 " + translator.T(locale, "prefs.short", nil), CallbackData: callbackPrefsPrefix + "short"}},
		{primaryExitButton(locale, translator)},
	}}
}

func languageInlineKeyboard(current string, translator Translator) *telegrammodels.InlineKeyboardMarkup {
	rows := make([][]telegrammodels.InlineKeyboardButton, 0, len(supportedLocales)/2+2)
	row := make([]telegrammodels.InlineKeyboardButton, 0, 2)
	for i, locale := range supportedLocales {
		button := telegrammodels.InlineKeyboardButton{
			Text:         locale.Flag,
			CallbackData: callbackPrefsPrefix + "lang:" + locale.Code,
		}
		if locale.Code == current {
			button.Style = buttonStyleSuccess
		}
		row = append(row, button)
		if len(row) == 2 || i == len(supportedLocales)-1 {
			rows = append(rows, row)
			row = make([]telegrammodels.InlineKeyboardButton, 0, 2)
		}
	}
	rows = append(rows, []telegrammodels.InlineKeyboardButton{primaryBackButton(), primaryExitButton(current, translator)})
	return &telegrammodels.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func timezoneInlineKeyboard(locale string, current string, translator Translator) *telegrammodels.InlineKeyboardMarkup {
	rows := make([][]telegrammodels.InlineKeyboardButton, 0, len(commonTimezones)/2+3)
	row := make([]telegrammodels.InlineKeyboardButton, 0, 2)
	for i, timezone := range commonTimezones {
		button := telegrammodels.InlineKeyboardButton{
			Text:         timezone,
			CallbackData: callbackPrefsPrefix + "tz:" + timezone,
		}
		if timezone == current {
			button.Style = buttonStyleSuccess
		}
		row = append(row, button)
		if len(row) == 2 || i == len(commonTimezones)-1 {
			rows = append(rows, row)
			row = make([]telegrammodels.InlineKeyboardButton, 0, 2)
		}
	}
	rows = append(rows, []telegrammodels.InlineKeyboardButton{primaryBackButton(), primaryExitButton(locale, translator)})
	return &telegrammodels.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func shortInlineKeyboard(locale string, current bool, translator Translator) *telegrammodels.InlineKeyboardMarkup {
	on := telegrammodels.InlineKeyboardButton{Text: translator.T(locale, "prefs.short_on", nil), CallbackData: callbackPrefsPrefix + "short:on"}
	off := telegrammodels.InlineKeyboardButton{Text: translator.T(locale, "prefs.short_off", nil), CallbackData: callbackPrefsPrefix + "short:off"}
	if current {
		on.Style = buttonStyleSuccess
	} else {
		off.Style = buttonStyleSuccess
	}
	return &telegrammodels.InlineKeyboardMarkup{InlineKeyboard: [][]telegrammodels.InlineKeyboardButton{
		{on, off},
		{primaryBackButton(), primaryExitButton(locale, translator)},
	}}
}

func primaryExitButton(locale string, translator Translator) telegrammodels.InlineKeyboardButton {
	return telegrammodels.InlineKeyboardButton{
		Text:         "✖️",
		CallbackData: callbackPrefsPrefix + "exit",
		Style:        buttonStylePrimary,
	}
}

func primaryBackButton() telegrammodels.InlineKeyboardButton {
	return telegrammodels.InlineKeyboardButton{
		Text:         "↩️",
		CallbackData: callbackPrefsPrefix + "back",
		Style:        buttonStylePrimary,
	}
}

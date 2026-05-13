package bot

import (
	"context"
	"fmt"

	telegram "github.com/go-telegram/bot"
	telegrammodels "github.com/go-telegram/bot/models"
)

type Messenger interface {
	SendMessage(ctx context.Context, chatID int64, text string, keyboard interface{}) (int, error)
	AnswerCallback(ctx context.Context, callbackID string) error
}

type TelegramMessenger struct {
	api *telegram.Bot
}

func NewTelegramMessenger(api *telegram.Bot) *TelegramMessenger {
	return &TelegramMessenger{api: api}
}

func (m *TelegramMessenger) SendMessage(ctx context.Context, chatID int64, text string, keyboard interface{}) (int, error) {
	params := &telegram.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: telegrammodels.ParseModeHTML,
	}
	if keyboard != nil {
		params.ReplyMarkup = keyboard
	} else {
		params.ReplyMarkup = mainMenuKeyboard()
	}

	msg, err := m.api.SendMessage(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("send message: %w", err)
	}
	if msg == nil {
		return 0, fmt.Errorf("send message: empty response")
	}
	return msg.ID, nil
}

func (m *TelegramMessenger) AnswerCallback(ctx context.Context, callbackID string) error {
	if callbackID == "" {
		return nil
	}
	if _, err := m.api.AnswerCallbackQuery(ctx, &telegram.AnswerCallbackQueryParams{CallbackQueryID: callbackID}); err != nil {
		return fmt.Errorf("answer callback: %w", err)
	}
	return nil
}

func (m *TelegramMessenger) SetCommands(ctx context.Context, translator Translator, locale string) error {
	_, err := m.api.SetMyCommands(ctx, &telegram.SetMyCommandsParams{
		Commands: []telegrammodels.BotCommand{
			{Command: "start", Description: translator.T(locale, "command.start", nil)},
			{Command: "events", Description: translator.T(locale, "command.events", nil)},
			{Command: "subscribe", Description: translator.T(locale, "command.subscribe", nil)},
			{Command: "unsubscribe", Description: translator.T(locale, "command.unsubscribe", nil)},
			{Command: "list", Description: translator.T(locale, "command.list", nil)},
			{Command: "status", Description: translator.T(locale, "command.status", nil)},
			{Command: "prefs", Description: translator.T(locale, "command.prefs", nil)},
			{Command: "help", Description: translator.T(locale, "command.help", nil)},
		},
	})
	if err != nil {
		return fmt.Errorf("set commands: %w", err)
	}
	return nil
}

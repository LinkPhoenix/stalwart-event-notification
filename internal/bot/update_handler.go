package bot

import (
	"context"
	"fmt"
	"strings"

	telegrammodels "github.com/go-telegram/bot/models"
)

func HandleUpdate(ctx context.Context, handler *Handler, update *telegrammodels.Update) error {
	if update == nil {
		return nil
	}
	if update.Message != nil {
		text := strings.TrimSpace(update.Message.Text)
		if text == "" {
			return nil
		}
		username := ""
		userID := ""
		languageCode := ""
		if update.Message.From != nil {
			userID = fmt.Sprintf("%d", update.Message.From.ID)
			languageCode = strings.TrimSpace(update.Message.From.LanguageCode)
			username = strings.TrimSpace(update.Message.From.Username)
			if username == "" {
				username = strings.TrimSpace(update.Message.From.FirstName)
			}
		}
		return handler.HandleMessage(ctx, update.Message.Chat.ID, userID, username, text, languageCode)
	}
	if update.CallbackQuery != nil {
		chatID, messageID, ok := callbackMessageTarget(update.CallbackQuery)
		if !ok {
			return nil
		}
		userID := fmt.Sprintf("%d", update.CallbackQuery.From.ID)
		languageCode := strings.TrimSpace(update.CallbackQuery.From.LanguageCode)
		return handler.HandleCallback(ctx, update.CallbackQuery.ID, chatID, messageID, userID, update.CallbackQuery.Data, languageCode)
	}
	return nil
}

func callbackMessageTarget(query *telegrammodels.CallbackQuery) (int64, int, bool) {
	if query == nil {
		return 0, 0, false
	}
	switch query.Message.Type {
	case telegrammodels.MaybeInaccessibleMessageTypeMessage:
		if query.Message.Message == nil {
			return 0, 0, false
		}
		return query.Message.Message.Chat.ID, query.Message.Message.ID, true
	case telegrammodels.MaybeInaccessibleMessageTypeInaccessibleMessage:
		if query.Message.InaccessibleMessage == nil {
			return 0, 0, false
		}
		return query.Message.InaccessibleMessage.Chat.ID, query.Message.InaccessibleMessage.MessageID, true
	default:
		return query.From.ID, 0, true
	}
}

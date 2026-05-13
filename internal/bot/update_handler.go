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
		if update.Message.From != nil {
			userID = fmt.Sprintf("%d", update.Message.From.ID)
			username = strings.TrimSpace(update.Message.From.Username)
			if username == "" {
				username = strings.TrimSpace(update.Message.From.FirstName)
			}
		}
		return handler.HandleMessage(ctx, update.Message.Chat.ID, userID, username, text)
	}
	if update.CallbackQuery != nil {
		chatID, ok := callbackChatID(update.CallbackQuery)
		if !ok {
			return nil
		}
		userID := fmt.Sprintf("%d", update.CallbackQuery.From.ID)
		return handler.HandleCallback(ctx, update.CallbackQuery.ID, chatID, userID, update.CallbackQuery.Data)
	}
	return nil
}

func callbackChatID(query *telegrammodels.CallbackQuery) (int64, bool) {
	if query == nil {
		return 0, false
	}
	switch query.Message.Type {
	case telegrammodels.MaybeInaccessibleMessageTypeMessage:
		if query.Message.Message == nil {
			return 0, false
		}
		return query.Message.Message.Chat.ID, true
	case telegrammodels.MaybeInaccessibleMessageTypeInaccessibleMessage:
		if query.Message.InaccessibleMessage == nil {
			return 0, false
		}
		return query.Message.InaccessibleMessage.Chat.ID, true
	default:
		return query.From.ID, true
	}
}

package notifications

import "context"

type notificationOnlyMessenger struct{}

func (notificationOnlyMessenger) SendMessage(ctx context.Context, chatID int64, text string, keyboard interface{}) (int, error) {
	return 1, nil
}

var _ Messenger = notificationOnlyMessenger{}

package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mymmrac/telego"
)

func sendMessage(ctx context.Context, bot *telego.Bot, chatID int64, text string) (*telego.Message, error) {
	message, err := bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text:   text,
	})
	if err != nil {
		slog.Error("Ошибка отправки сообщения",
			slog.Int64("chat_id", chatID),
			slog.Any("error", err))
	}
	return message, err
}

// slogLogger адаптирует интерфейс логгера telego под стандартный slog проекта,
// чтобы сообщения библиотеки попадали в тот же JSON-поток логов.
type slogLogger struct{}

func (slogLogger) Debugf(format string, args ...any) {
	slog.Debug(fmt.Sprintf(format, args...))
}

func (slogLogger) Errorf(format string, args ...any) {
	slog.Error(fmt.Sprintf(format, args...))
}

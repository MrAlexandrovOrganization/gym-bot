package main

import (
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	message, err := bot.Send(msg)
	if err != nil {
		slog.Error("Ошибка отправки сообщения",
			slog.Int64("chat_id", chatID),
			slog.Any("error", err))
	}
	return message, err
}

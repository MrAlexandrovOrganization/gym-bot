package main

import (
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CommandHandler func(app *App, msg *tgbotapi.Message)

var commands = map[string]CommandHandler{
	"start":    handleStart,
	"help":     handleHelp,
	"newpoll":  handleNewPoll,
	"findpoll": handleFindCurrentPoll,
}

func NewBot(token, apiEndpoint string) (*tgbotapi.BotAPI, error) {
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is empty")
	}
	if apiEndpoint == "" {
		return tgbotapi.NewBotAPI(token)
	}
	return tgbotapi.NewBotAPIWithAPIEndpoint(token, apiEndpoint)
}

// handleCommand обрабатывает команды бота
func (a *App) handleCommand(msg *tgbotapi.Message) {
	handler, ok := commands[msg.Command()]
	if !ok {
		sendMessage(a.bot, msg.Chat.ID, handleUnknownCommandText)
		return
	}
	handler(a, msg)
}

// handleStart обрабатывает команду /start
func handleStart(app *App, msg *tgbotapi.Message) {
	sendMessage(app.bot, msg.Chat.ID, handleStartText)
}

// handleHelp обрабатывает команду /help
func handleHelp(app *App, msg *tgbotapi.Message) {
	sendMessage(app.bot, msg.Chat.ID, handleHelpText)
}

func checkWeekPoll(db DB, chatID int64, weekNumber int, year int) (bool, error) {
	hasWeekPoll, err := db.HasWeekPoll(chatID, weekNumber, year)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки опроса: %w", err)
	}
	return hasWeekPoll, nil
}

func createPoll(chatID int64) tgbotapi.SendPollConfig {
	return tgbotapi.SendPollConfig{
		BaseChat:              tgbotapi.BaseChat{ChatID: chatID},
		Question:              pollQuestion,
		Options:               weekdays,
		IsAnonymous:           false,
		AllowsMultipleAnswers: true,
	}
}

func (a *App) newPollImpl(chatID int64, weekNumber, year int) {
	a.pollMu.Lock()
	defer a.pollMu.Unlock()

	hasWeekPoll, err := checkWeekPoll(a.db, chatID, weekNumber, year)
	if err != nil {
		slog.Error("Ошибка проверки опроса", slog.Any("error", err))
		sendMessage(a.bot, chatID, errorCheckPollText)
		return
	}
	if hasWeekPoll {
		a.findPollImpl(chatID, weekNumber, year)
		return
	}

	pollMsg, err := a.bot.Send(createPoll(chatID))
	if err != nil {
		slog.Error("Ошибка создания опроса", slog.Any("error", err))
		sendMessage(a.bot, chatID, errorCreatePollText)
		return
	}

	go func() {
		pinChatMessageConfig := tgbotapi.PinChatMessageConfig{
			ChatID:              pollMsg.Chat.ID,
			MessageID:           pollMsg.MessageID,
			DisableNotification: true,
		}
		_, err = a.bot.Request(pinChatMessageConfig)
		if err != nil {
			slog.Error("Ошибка закрепления сообщения",
				slog.Int("message_id", pollMsg.MessageID),
				slog.Any("error", err))
			return
		}
	}()

	if err = a.db.SavePoll(chatID, pollMsg.MessageID, pollMsg.Poll.ID, weekNumber, year); err != nil {
		slog.Error("Ошибка сохранения опроса", slog.Any("error", err))
		sendMessage(a.bot, chatID, errorSavePollText)
		return
	}

	sendMessage(a.bot, chatID, successSavePollText)
}

func handleNewPoll(app *App, msg *tgbotapi.Message) {
	weekNumber, year := getCurrentWeekAndYear()
	app.newPollImpl(msg.Chat.ID, weekNumber, year)
}

func (a *App) findPollImpl(chatID int64, weekNumber, year int) {
	poll, err := a.db.GetWeekPoll(chatID, weekNumber, year)
	if err != nil {
		slog.Error("Ошибка получения опроса", slog.Any("error", err))
		sendMessage(a.bot, chatID, errorSearchPollText)
		return
	}

	if poll == nil {
		sendMessage(a.bot, chatID, errorNoPollText)
		return
	}

	reply := tgbotapi.NewMessage(chatID, currentPollText)
	reply.ReplyToMessageID = poll.MessageID

	_, err = a.bot.Send(reply)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "message to reply not found") ||
			strings.Contains(errMsg, "replied message not found") {
			sendMessage(a.bot, chatID, errorPollUnavailableText)
			return
		}
		slog.Error("Ошибка отправки ответа", slog.Any("error", err))
		sendMessage(a.bot, chatID, errorSendAnswerText)
	}
}

// handleFindCurrentPoll обрабатывает команду /findpoll
func handleFindCurrentPoll(app *App, msg *tgbotapi.Message) {
	weekNumber, year := getCurrentWeekAndYear()
	app.findPollImpl(msg.Chat.ID, weekNumber, year)
}

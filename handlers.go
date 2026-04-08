package main

import (
	"fmt"
	"log"
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

func NewBot(token string) (*tgbotapi.BotAPI, error) {
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is empty")
	}
	return tgbotapi.NewBotAPI(token)
}

// handleCommand обрабатывает команды бота
func (a *App) handleCommand(msg *tgbotapi.Message) {
	handler, ok := commands[msg.Command()]
	if !ok {
		a.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, handleUnknownCommandText))
		return
	}
	handler(a, msg)
}

// handleStart обрабатывает команду /start
func handleStart(app *App, msg *tgbotapi.Message) {
	app.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, handleStartText))
}

// handleHelp обрабатывает команду /help
func handleHelp(app *App, msg *tgbotapi.Message) {
	app.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, handleHelpText))
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
	hasWeekPoll, err := checkWeekPoll(a.db, chatID, weekNumber, year)
	if err != nil {
		log.Printf("Ошибка проверки опроса: %v", err)
		a.bot.Send(tgbotapi.NewMessage(chatID, errorCheckPollText))
		return
	}
	if hasWeekPoll {
		a.findPollImpl(chatID, weekNumber, year)
		return
	}

	pollMsg, err := a.bot.Send(createPoll(chatID))
	if err != nil {
		log.Printf("Ошибка создания опроса: %v", err)
		a.bot.Send(tgbotapi.NewMessage(chatID, errorCreatePollText))
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
			log.Printf("Ошибка закрепления сообщения: %v", err)
			return
		}
	}()

	if err = a.db.SavePoll(chatID, pollMsg.MessageID, pollMsg.Poll.ID, weekNumber, year); err != nil {
		log.Printf("Ошибка сохранения опроса: %v", err)
		a.bot.Send(tgbotapi.NewMessage(chatID, errorSavePollText))
		return
	}

	a.bot.Send(tgbotapi.NewMessage(chatID, successSavePollText))
}

func handleNewPoll(app *App, msg *tgbotapi.Message) {
	weekNumber, year := getCurrentWeekAndYear()
	app.newPollImpl(msg.Chat.ID, weekNumber, year)
}

func (a *App) findPollImpl(chatID int64, weekNumber, year int) {
	poll, err := a.db.GetWeekPoll(chatID, weekNumber, year)
	if err != nil {
		log.Printf("Ошибка получения опроса: %v", err)
		a.bot.Send(tgbotapi.NewMessage(chatID, errorSearchPollText))
		return
	}

	if poll == nil {
		a.bot.Send(tgbotapi.NewMessage(chatID, errorNoPollText))
		return
	}

	reply := tgbotapi.NewMessage(chatID, currentPollText)
	reply.ReplyToMessageID = poll.MessageID

	_, err = a.bot.Send(reply)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "message to reply not found") ||
			strings.Contains(errMsg, "replied message not found") {
			a.bot.Send(tgbotapi.NewMessage(chatID, errorPollUnavailableText))
			return
		}
		log.Printf("Ошибка отправки ответа: %v", err)
		a.bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf(errorSendAnswerText+": %v", err)))
	}
}

// handleFindCurrentPoll обрабатывает команду /findpoll
func handleFindCurrentPoll(app *App, msg *tgbotapi.Message) {
	weekNumber, year := getCurrentWeekAndYear()
	app.findPollImpl(msg.Chat.ID, weekNumber, year)
}

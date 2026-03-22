package main

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CommandHandler func(bot *tgbotapi.BotAPI, db DB, msg *tgbotapi.Message)

var commands = map[string]CommandHandler{
	"start":    handleStart,
	"help":     handleHelp,
	"newpoll":  handleNewPoll,
	"findpoll": handleFindCurrentPoll,
}

// handleCommand обрабатывает команды бота
func handleCommand(bot *tgbotapi.BotAPI, db DB, msg *tgbotapi.Message) {
	handler, ok := commands[msg.Command()]
	if !ok {
		reply := tgbotapi.NewMessage(msg.Chat.ID, handleUnknownCommandText)
		bot.Send(reply)
		return
	}
	handler(bot, db, msg)
}

// handleStart обрабатывает команду /start
func handleStart(bot *tgbotapi.BotAPI, db DB, msg *tgbotapi.Message) {
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, handleStartText))
}

// handleHelp обрабатывает команду /help
func handleHelp(bot *tgbotapi.BotAPI, db DB, msg *tgbotapi.Message) {
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, handleHelpText))
}

func checkWeekPoll(db DB, chatID int64, weekNumber int, year int) (bool, error) {
	hasWeekPoll, err := db.HasWeekPoll(chatID, weekNumber, year)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки опроса: %w", err)
	}
	return hasWeekPoll, nil
}

// checkCurrentWeekPoll — чистая логика, без Telegram
func checkCurrentWeekPoll(db DB, chatID int64) (bool, error) {
	weekNumber, year := getCurrentWeekAndYear()

	hasWeekPoll, err := checkWeekPoll(db, chatID, weekNumber, year)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки опроса: %w", err)
	}
	return hasWeekPoll, nil
}

func createPoll(chatID int64) tgbotapi.SendPollConfig {
	return tgbotapi.SendPollConfig{
		BaseChat:              tgbotapi.BaseChat{ChatID: chatID},
		Question:              poolQuestion,
		Options:               weekdays,
		IsAnonymous:           false,
		AllowsMultipleAnswers: true,
	}
}

func newPollImpl(bot *tgbotapi.BotAPI, db DB, chatID int64, weekNumber, year int) {
	hasWeekPoll, err := checkWeekPoll(db, chatID, weekNumber, year)
	if err != nil {
		log.Printf("Ошибка проверки опроса: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, errorCheckPoolText))
		return
	}
	if hasWeekPoll {
		findPollImpl(bot, db, chatID, weekNumber, year)
		return
	}

	pollMsg, err := bot.Send(createPoll(chatID))
	if err != nil {
		log.Printf("Ошибка создания опроса: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, errorCreatePollText))
		return
	}

	if err = db.SavePoll(chatID, pollMsg.MessageID, pollMsg.Poll.ID, weekNumber, year); err != nil {
		log.Printf("Ошибка сохранения опроса: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, errorSavePollText))
		return
	}

	bot.Send(tgbotapi.NewMessage(chatID, successSavePollText))
}

func handleNewPoll(bot *tgbotapi.BotAPI, db DB, msg *tgbotapi.Message) {
	weekNumber, year := getCurrentWeekAndYear()

	newPollImpl(bot, db, msg.Chat.ID, weekNumber, year)
}

func findPollImpl(bot *tgbotapi.BotAPI, db DB, chatID int64, weekNuber, year int) {
	// Получаем опрос текущей недели
	poll, err := db.GetWeekPoll(chatID, weekNuber, year)
	if err != nil {
		log.Printf("Ошибка получения опроса: %v", err)
		reply := tgbotapi.NewMessage(chatID, errorSearchPollText)
		bot.Send(reply)
		return
	}

	if poll == nil {
		reply := tgbotapi.NewMessage(chatID, errorNoPollText)
		bot.Send(reply)
		return
	}

	// Отправляем ответ на сообщение с опросом
	reply := tgbotapi.NewMessage(chatID, currentPollText)
	reply.ReplyToMessageID = poll.MessageID

	_, err = bot.Send(reply)
	if err != nil {
		// Если сообщение не найдено (удалено или недоступно)
		if err.Error() == "Bad Request: message to reply not found" ||
			err.Error() == "Bad Request: replied message not found" {
			errorReply := tgbotapi.NewMessage(chatID, errorPollUnavailableText)
			bot.Send(errorReply)
			return
		}
		log.Printf("Ошибка отправки ответа: %v", err)
		errorReply := tgbotapi.NewMessage(chatID, fmt.Sprintf(errorSendAnswerText+": %v", err))
		bot.Send(errorReply)
	}
}

// handleFindPoll обрабатывает команду /findpoll
func handleFindCurrentPoll(bot *tgbotapi.BotAPI, db DB, msg *tgbotapi.Message) {
	weekNumber, year := getCurrentWeekAndYear()

	findPollImpl(bot, db, msg.Chat.ID, weekNumber, year)
}

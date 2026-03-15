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
	"findpoll": handleFindPoll,
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

// checkWeekPoll — чистая логика, без Telegram
func checkWeekPoll(db DB, chatID int64) (bool, error) {
	hasWeekPoll, err := db.HasWeekPoll(chatID)
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

func handleNewPollImpl(bot *tgbotapi.BotAPI, db DB, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID

	hasWeekPoll, err := checkWeekPoll(db, chatID)
	if err != nil {
		log.Printf("Ошибка проверки опроса: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, errorCheckPoolText))
		return
	}
	if hasWeekPoll {
		bot.Send(tgbotapi.NewMessage(chatID, errorAlreadyExistsText))
		return
	}

	pollMsg, err := bot.Send(createPoll(chatID))
	if err != nil {
		log.Printf("Ошибка создания опроса: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, errorCreatePollText))
		return
	}

	if err = db.SavePoll(chatID, pollMsg.MessageID, pollMsg.Poll.ID); err != nil {
		log.Printf("Ошибка сохранения опроса: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, errorSavePollText))
		return
	}

	bot.Send(tgbotapi.NewMessage(chatID, successSavePollText))
}

// handleNewPoll обрабатывает команду /newpoll
func handleNewPoll(bot *tgbotapi.BotAPI, db DB, msg *tgbotapi.Message) {
	handleNewPollImpl(bot, db, msg)
}

// handleFindPoll обрабатывает команду /findpoll
func handleFindPoll(bot *tgbotapi.BotAPI, db DB, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID

	// Получаем опрос текущей недели
	poll, err := db.GetCurrentWeekPoll(chatID)
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

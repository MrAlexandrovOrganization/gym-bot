package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
)

type CommandHandler func(app *App, ctx context.Context, msg *telego.Message)

var commands = map[string]CommandHandler{
	"start":    handleStart,
	"help":     handleHelp,
	"newpoll":  handleNewPoll,
	"findpoll": handleFindCurrentPoll,
}

// NewBot создаёт клиента Telegram Bot API. Пустой apiServerURL означает
// официальный api.telegram.org, иначе используется указанный локальный сервер.
func NewBot(token, apiServerURL string) (*telego.Bot, error) {
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is empty")
	}

	opts := []telego.BotOption{telego.WithLogger(slogLogger{})}
	if apiServerURL != "" {
		opts = append(opts, telego.WithAPIServer(apiServerURL))
	}

	return telego.NewBot(token, opts...)
}

// handleCommand обрабатывает команды бота
func (a *App) handleCommand(ctx context.Context, cmd string, msg *telego.Message) {
	handler, ok := commands[cmd]
	if !ok {
		sendMessage(ctx, a.bot, msg.Chat.ID, handleUnknownCommandText)
		return
	}
	handler(a, ctx, msg)
}

// handleStart обрабатывает команду /start
func handleStart(app *App, ctx context.Context, msg *telego.Message) {
	sendMessage(ctx, app.bot, msg.Chat.ID, handleStartText)
}

// handleHelp обрабатывает команду /help
func handleHelp(app *App, ctx context.Context, msg *telego.Message) {
	sendMessage(ctx, app.bot, msg.Chat.ID, handleHelpText)
}

func checkWeekPoll(db DB, chatID int64, weekNumber int, year int) (bool, error) {
	hasWeekPoll, err := db.HasWeekPoll(chatID, weekNumber, year)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки опроса: %w", err)
	}
	return hasWeekPoll, nil
}

func createPoll(chatID int64) *telego.SendPollParams {
	options := make([]telego.InputPollOption, 0, len(weekdays))
	for _, day := range weekdays {
		options = append(options, telegoutil.PollOption(day))
	}

	return telegoutil.Poll(telego.ChatID{ID: chatID}, pollQuestion, options...).
		WithIsAnonymous(false).
		WithAllowsMultipleAnswers()
}

func (a *App) newPollImpl(ctx context.Context, chatID int64, weekNumber, year int) {
	a.pollMu.Lock()
	defer a.pollMu.Unlock()

	hasWeekPoll, err := checkWeekPoll(a.db, chatID, weekNumber, year)
	if err != nil {
		slog.Error("Ошибка проверки опроса", slog.Any("error", err))
		sendMessage(ctx, a.bot, chatID, errorCheckPollText)
		return
	}
	if hasWeekPoll {
		a.findPollImpl(ctx, chatID, weekNumber, year)
		return
	}

	pollMsg, err := a.bot.SendPoll(ctx, createPoll(chatID))
	if err != nil {
		slog.Error("Ошибка создания опроса", slog.Any("error", err))
		sendMessage(ctx, a.bot, chatID, errorCreatePollText)
		return
	}

	go func() {
		err := a.bot.PinChatMessage(ctx, &telego.PinChatMessageParams{
			ChatID:              telego.ChatID{ID: pollMsg.Chat.ID},
			MessageID:           pollMsg.MessageID,
			DisableNotification: true,
		})
		if err != nil {
			slog.Error("Ошибка закрепления сообщения",
				slog.Int("message_id", pollMsg.MessageID),
				slog.Any("error", err))
			return
		}
	}()

	if err = a.db.SavePoll(chatID, pollMsg.MessageID, pollMsg.Poll.ID, weekNumber, year); err != nil {
		slog.Error("Ошибка сохранения опроса", slog.Any("error", err))
		sendMessage(ctx, a.bot, chatID, errorSavePollText)
		return
	}

	sendMessage(ctx, a.bot, chatID, successSavePollText)
}

func handleNewPoll(app *App, ctx context.Context, msg *telego.Message) {
	weekNumber, year := getCurrentWeekAndYear()
	app.newPollImpl(ctx, msg.Chat.ID, weekNumber, year)
}

func (a *App) findPollImpl(ctx context.Context, chatID int64, weekNumber, year int) {
	poll, err := a.db.GetWeekPoll(chatID, weekNumber, year)
	if err != nil {
		slog.Error("Ошибка получения опроса", slog.Any("error", err))
		sendMessage(ctx, a.bot, chatID, errorSearchPollText)
		return
	}

	if poll == nil {
		sendMessage(ctx, a.bot, chatID, errorNoPollText)
		return
	}

	reply := &telego.SendMessageParams{
		ChatID:          telego.ChatID{ID: chatID},
		Text:            currentPollText,
		ReplyParameters: &telego.ReplyParameters{MessageID: poll.MessageID},
	}

	_, err = a.bot.SendMessage(ctx, reply)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "message to reply not found") ||
			strings.Contains(errMsg, "replied message not found") {
			sendMessage(ctx, a.bot, chatID, errorPollUnavailableText)
			return
		}
		slog.Error("Ошибка отправки ответа", slog.Any("error", err))
		sendMessage(ctx, a.bot, chatID, errorSendAnswerText)
	}
}

// handleFindCurrentPoll обрабатывает команду /findpoll
func handleFindCurrentPoll(app *App, ctx context.Context, msg *telego.Message) {
	weekNumber, year := getCurrentWeekAndYear()
	app.findPollImpl(ctx, msg.Chat.ID, weekNumber, year)
}

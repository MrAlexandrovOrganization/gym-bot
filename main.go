package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"

	"github.com/mrralexandrov/gym-bot/logx"
)

type App struct {
	bot           *tgbotapi.BotAPI
	db            DB
	allowedChatID int64
	pollMu        sync.Mutex
}

func getAllowedChatID() (int64, error) {
	s := os.Getenv("ALLOWED_CHAT_ID")
	if s == "" {
		return 0, fmt.Errorf("ALLOWED_CHAT_ID не установлен")
	}
	return strconv.ParseInt(s, 10, 64)
}

// apiEndpoint builds the go-telegram-bot-api endpoint format string for the
// local Telegram Bot API server, or "" to use the default api.telegram.org.
func apiEndpoint() string {
	base := strings.TrimRight(os.Getenv("TELEGRAM_LOCAL_API_URL"), "/")
	if base == "" {
		return ""
	}
	return base + "/bot%s/%s"
}

func init() {
	// .env загружается до установки логгера, чтобы LOG_LEVEL из .env учитывался.
	err := godotenv.Load()
	logx.Setup("gym-bot", os.Getenv("TELEGRAM_BOT_TOKEN"))
	if err != nil {
		slog.Warn("Файл .env не найден, используются системные переменные окружения")
	}
}

func newApp() (*App, error) {
	app := &App{}
	g := new(errgroup.Group)

	g.Go(func() (err error) {
		app.bot, err = NewBot(os.Getenv("TELEGRAM_BOT_TOKEN"), apiEndpoint())
		if err != nil {
			return err
		}
		slog.Info("Авторизован как", slog.String("username", app.bot.Self.UserName))
		return nil
	})

	g.Go(func() (err error) {
		app.db, err = NewDatabase(os.Getenv("DATABASE_URL"))
		if err != nil {
			return err
		}
		slog.Info("Успешное подключение к PostgreSQL")
		return nil
	})

	g.Go(func() (err error) {
		app.allowedChatID, err = getAllowedChatID()
		if err != nil {
			return err
		}
		slog.Info("Бот работает только с чатом", slog.Int64("chat_id", app.allowedChatID))
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return app, nil
}

func (a *App) startPolling() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := a.bot.GetUpdatesChan(u)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Завершение работы бота...")
		a.bot.StopReceivingUpdates()
	}()

	slog.Info("✅ Бот запущен и готов к работе!")

	for update := range updates {
		go func() {
			if update.Message == nil {
				return
			}

			if update.Message.Chat.ID != a.allowedChatID {
				slog.Warn("Игнорируем сообщение из неразрешённого чата",
					slog.Int64("chat_id", update.Message.Chat.ID),
					slog.Int64("allowed_chat_id", a.allowedChatID))
				return
			}

			if update.Message.IsCommand() {
				a.handleCommand(update.Message)
			}
		}()
	}
}

func main() {
	app, err := newApp()
	if err != nil {
		slog.Error("Ошибка инициализации приложения", slog.Any("error", err))
		os.Exit(1)
	}
	defer app.db.Close()

	startCron("0 22 * * 0", func() {
		slog.Info("[cron] Запуск автоматического создания опроса...")
		nextDate := time.Now().In(moscow).AddDate(0, 0, 1)
		week, year := getWeekAndYear(nextDate)
		app.newPollImpl(app.allowedChatID, week, year)
	})
	app.startPolling()
}

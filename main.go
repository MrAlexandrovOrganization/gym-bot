package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
	"golang.org/x/sync/errgroup"

	"github.com/mrralexandrov/gym-bot/logx"
)

type App struct {
	bot           *telego.Bot
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

// apiServerURL возвращает базовый URL локального Telegram Bot API-сервера
// (telego сам добавляет к нему /bot<token>/<method>), либо "" для
// использования api.telegram.org.
func apiServerURL() string {
	return strings.TrimRight(os.Getenv("TELEGRAM_LOCAL_API_URL"), "/")
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
		app.bot, err = NewBot(os.Getenv("TELEGRAM_BOT_TOKEN"), apiServerURL())
		if err != nil {
			return err
		}
		me, err := app.bot.GetMe(context.Background())
		if err != nil {
			return fmt.Errorf("не удалось авторизоваться в Telegram: %w", err)
		}
		slog.Info("Авторизован как", slog.String("username", me.Username))
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updates, err := a.bot.UpdatesViaLongPolling(ctx, &telego.GetUpdatesParams{Timeout: 60})
	if err != nil {
		slog.Error("Ошибка запуска long polling", slog.Any("error", err))
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Завершение работы бота...")
		cancel()
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

			cmd, _, _ := telegoutil.ParseCommand(update.Message.Text)
			if cmd == "" {
				return
			}

			a.handleCommand(ctx, cmd, update.Message)
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
		app.newPollImpl(context.Background(), app.allowedChatID, week, year)
	})
	app.startPolling()
}

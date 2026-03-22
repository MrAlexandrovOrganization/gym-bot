package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

type App struct {
	bot           *tgbotapi.BotAPI
	db            DB
	allowedChatID int64
}

func getAllowedChatID() (int64, error) {
	s := os.Getenv("ALLOWED_CHAT_ID")
	if s == "" {
		return 0, fmt.Errorf("ALLOWED_CHAT_ID не установлен")
	}
	return strconv.ParseInt(s, 10, 64)
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используются системные переменные окружения")
	}
}

func newApp() (*App, error) {
	app := &App{}
	g := new(errgroup.Group)

	g.Go(func() (err error) {
		app.bot, err = NewBot(os.Getenv("TELEGRAM_BOT_TOKEN"))
		if err != nil {
			return err
		}
		log.Printf("Авторизован как @%s", app.bot.Self.UserName)
		return nil
	})

	g.Go(func() (err error) {
		app.db, err = NewDatabase(os.Getenv("DATABASE_URL"))
		if err != nil {
			return err
		}
		log.Println("Успешное подключение к PostgreSQL")
		return nil
	})

	g.Go(func() (err error) {
		app.allowedChatID, err = getAllowedChatID()
		if err != nil {
			return err
		}
		log.Printf("Бот работает только с чатом ID: %d", app.allowedChatID)
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
		log.Println("Завершение работы бота...")
		a.bot.StopReceivingUpdates()
	}()

	log.Println("✅ Бот запущен и готов к работе!")

	for update := range updates {
		go func() {
			if update.Message == nil {
				return
			}

			if update.Message.Chat.ID != a.allowedChatID {
				log.Printf("Игнорируем сообщение из неразрешённого чата: %d (разрешён: %d)",
					update.Message.Chat.ID, a.allowedChatID)
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
		log.Fatal(err)
	}
	defer app.db.Close()

	startCron("0 22 * * 0", func() {
		log.Println("[cron] Запуск автоматического создания опроса...")
		nextDate := time.Now().AddDate(0, 0, 1)
		week, year := getWeekAndYear(nextDate)
		app.newPollImpl(app.allowedChatID, week, year)
	})
	app.startPolling()
}

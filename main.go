package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

var (
	bot           *tgbotapi.BotAPI
	db            *Database
	allowedChatID int64
)

func createBot() (bot *tgbotapi.BotAPI, err error) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN не установлен")
	}

	bot, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return bot, err
}

func createDatabase() (db *Database, err error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL не установлен")
	}

	db, err = NewDatabase(databaseURL)
	if err != nil {
		return nil, err
	}

	return db, err
}

func getAllowedChatID() (allowedChatID int64, err error) {
	allowedChatIDStr := os.Getenv("ALLOWED_CHAT_ID")
	if allowedChatIDStr == "" {
		return 0, fmt.Errorf("ALLOWED_CHAT_ID не установлен")
	}

	allowedChatID, err = strconv.ParseInt(allowedChatIDStr, 10, 64)
	if err != nil {
		return 0, err
	}

	return allowedChatID, nil
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используются системные переменные окружения")
	}
}

func prepare() (err error) {
	g := new(errgroup.Group)

	g.Go(func() (err error) {
		bot, err = createBot()
		if err != nil {
			return err
		}
		log.Printf("Авторизован как @%s", bot.Self.UserName)
		return nil
	})

	g.Go(func() (err error) {
		db, err = createDatabase()
		if err != nil {
			return err
		}
		log.Println("Успешное подключение к PostgreSQL")
		return nil
	})

	g.Go(func() (err error) {
		allowedChatID, err = getAllowedChatID()
		if err != nil {
			return err
		}
		log.Printf("Бот работает только с чатом ID: %d", allowedChatID)
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

func startPolling() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Завершение работы бота...")
		bot.StopReceivingUpdates()
		os.Exit(0)
	}()

	log.Println("✅ Бот запущен и готов к работе!")

	for update := range updates {
		go func() {
			if update.Message == nil {
				return
			}

			if update.Message.Chat.ID != allowedChatID {
				log.Printf("Игнорируем сообщение из неразрешённого чата: %d (разрешён: %d)",
					update.Message.Chat.ID, allowedChatID)
				return
			}

			if update.Message.IsCommand() {
				handleCommand(bot, db, update.Message)
			}
		}()
	}
}

func main() {
	if err := prepare(); err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	startCron("0 8 * * 1", func() {
		log.Println("[cron] Запуск автоматического создания опроса...")
		handleNewPollImpl(bot, db, allowedChatID)
	})
	startPolling()
}

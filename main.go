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
)

var (
	// Дни недели для опроса
	weekdays = []string{
		"Понедельник",
		"Вторник",
		"Среда",
		"Четверг",
		"Пятница",
		"Суббота",
		"Воскресенье",
		"Не могу",
	}

	// ID разрешённого чата (будет установлен из env)
	allowedChatID int64
)

func main() {
	// Загружаем переменные окружения из .env файла
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используются системные переменные окружения")
	}

	// Получаем токен бота
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("Ошибка: TELEGRAM_BOT_TOKEN не установлен")
	}

	// Получаем ID разрешённого чата
	allowedChatIDStr := os.Getenv("ALLOWED_CHAT_ID")
	if allowedChatIDStr == "" {
		log.Fatal("Ошибка: ALLOWED_CHAT_ID не установлен")
	}

	var err error
	allowedChatID, err = strconv.ParseInt(allowedChatIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Ошибка: ALLOWED_CHAT_ID должен быть числом: %v", err)
	}

	// Получаем строку подключения к базе данных
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("Ошибка: DATABASE_URL не установлен")
	}

	// Создаём бота
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	log.Printf("Авторизован как @%s", bot.Self.UserName)
	log.Printf("Бот работает только с чатом ID: %d", allowedChatID)

	// Подключаемся к базе данных PostgreSQL
	db, err := NewDatabase(databaseURL)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	log.Println("Успешное подключение к PostgreSQL")

	// Настраиваем получение обновлений
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
		db.Close()
		os.Exit(0)
	}()

	log.Println("✅ Бот запущен и готов к работе!")

	// Обрабатываем обновления
	for update := range updates {
		if update.Message == nil {
			continue
		}

		// Проверяем, что сообщение из разрешённого чата
		if update.Message.Chat.ID != allowedChatID {
			log.Printf("Игнорируем сообщение из неразрешённого чата: %d (разрешён: %d)",
				update.Message.Chat.ID, allowedChatID)
			continue
		}

		// Обрабатываем команды
		if update.Message.IsCommand() {
			handleCommand(bot, db, update.Message)
		}
	}
}

// handleCommand обрабатывает команды бота
func handleCommand(bot *tgbotapi.BotAPI, db *Database, msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start":
		handleStart(bot, msg)
	case "help":
		handleHelp(bot, msg)
	case "newpoll":
		handleNewPoll(bot, db, msg)
	case "findpoll":
		handleFindPoll(bot, db, msg)
	default:
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Неизвестная команда. Используйте /help для списка доступных команд.")
		bot.Send(reply)
	}
}

// handleStart обрабатывает команду /start
func handleStart(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	text := `👋 Привет! Я бот для организации посещения спортзала.

📋 Доступные команды:

/newpoll - Создать новый опрос на неделю
/findpoll - Найти опрос текущей недели
/help - Показать эту справку

Просто добавь меня в групповой чат и используй команды!`

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	bot.Send(reply)
}

// handleHelp обрабатывает команду /help
func handleHelp(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	text := `📋 Список команд:

/newpoll - Создать новый опрос для выбора дней посещения зала
   • Можно создать только один опрос на неделю
   • Опрос содержит все дни недели + вариант "Не могу"
   • Опрос не анонимный

/findpoll - Отправить ответ на опрос текущей недели
   • Помогает быстро найти опрос в истории сообщений

/help - Показать это сообщение`

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	bot.Send(reply)
}

// handleNewPoll обрабатывает команду /newpoll
func handleNewPoll(bot *tgbotapi.BotAPI, db *Database, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID

	// Проверяем, есть ли уже опрос на эту неделю
	hasWeekPoll, err := db.HasWeekPoll(chatID)
	if err != nil {
		log.Printf("Ошибка проверки опроса: %v", err)
		reply := tgbotapi.NewMessage(chatID, "❌ Произошла ошибка при проверке опроса.")
		bot.Send(reply)
		return
	}

	if hasWeekPoll {
		reply := tgbotapi.NewMessage(chatID, "⚠️ На текущую неделю уже создан опрос! Используйте /findpoll, чтобы найти его.")
		bot.Send(reply)
		return
	}

	// Создаём опрос
	poll := tgbotapi.SendPollConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID: chatID,
		},
		Question:              "🏋️ Когда идёте в зал на этой неделе?",
		Options:               weekdays,
		IsAnonymous:           false,
		AllowsMultipleAnswers: true,
	}

	pollMsg, err := bot.Send(poll)
	if err != nil {
		log.Printf("Ошибка создания опроса: %v", err)
		reply := tgbotapi.NewMessage(chatID, "❌ Произошла ошибка при создании опроса.")
		bot.Send(reply)
		return
	}

	// Сохраняем опрос в базу данных
	err = db.SavePoll(chatID, pollMsg.MessageID, pollMsg.Poll.ID)
	if err != nil {
		log.Printf("Ошибка сохранения опроса: %v", err)
		reply := tgbotapi.NewMessage(chatID, "⚠️ Опрос создан, но не удалось сохранить его в базу данных.")
		bot.Send(reply)
		return
	}

	// Отправляем подтверждение
	reply := tgbotapi.NewMessage(chatID, "✅ Опрос создан! Выберите дни, когда планируете пойти в зал.\n\nИспользуйте /findpoll, чтобы быстро найти этот опрос позже.")
	bot.Send(reply)
}

// handleFindPoll обрабатывает команду /findpoll
func handleFindPoll(bot *tgbotapi.BotAPI, db *Database, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID

	// Получаем опрос текущей недели
	poll, err := db.GetCurrentWeekPoll(chatID)
	if err != nil {
		log.Printf("Ошибка получения опроса: %v", err)
		reply := tgbotapi.NewMessage(chatID, "❌ Произошла ошибка при поиске опроса.")
		bot.Send(reply)
		return
	}

	if poll == nil {
		reply := tgbotapi.NewMessage(chatID, "❌ Опрос на текущую неделю ещё не создан.\n\nИспользуйте /newpoll, чтобы создать новый опрос.")
		bot.Send(reply)
		return
	}

	// Отправляем ответ на сообщение с опросом
	reply := tgbotapi.NewMessage(chatID, "👆 Вот опрос на текущую неделю!")
	reply.ReplyToMessageID = poll.MessageID

	_, err = bot.Send(reply)
	if err != nil {
		// Если сообщение не найдено (удалено или недоступно)
		if err.Error() == "Bad Request: message to reply not found" ||
			err.Error() == "Bad Request: replied message not found" {
			errorReply := tgbotapi.NewMessage(chatID, "⚠️ Опрос был удалён или недоступен.\n\nСоздайте новый опрос с помощью /newpoll (но только со следующей недели, так как опрос на текущую неделю уже был зарегистрирован в базе).")
			bot.Send(errorReply)
			return
		}
		log.Printf("Ошибка отправки ответа: %v", err)
		errorReply := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Произошла ошибка при отправке ответа: %v", err))
		bot.Send(errorReply)
	}
}

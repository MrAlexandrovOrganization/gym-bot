# Gym Bot

Telegram бот на Go для организации еженедельных опросов о посещении спортзала в групповых чатах.

## Стек

- **Go 1.21** — основной язык
- **PostgreSQL 15** — хранение опросов
- **Docker + docker-compose** — деплой и локальная разработка
- **github.com/go-telegram-bot-api/telegram-bot-api/v5** — Telegram Bot API
- **github.com/lib/pq** — PostgreSQL драйвер
- **github.com/joho/godotenv** — загрузка `.env`

## Структура проекта

```
main.go        # Точка входа, обработка команд бота (/start, /help, /newpoll, /findpoll)
database.go    # Подключение к PostgreSQL, CRUD для таблицы polls
```

## Команды разработки

```bash
# Запустить через Docker (основной способ)
make up

# Пересобрать образы и перезапустить
make rebuild

# Логи бота
make logs-bot

# Запустить локально (нужен PostgreSQL)
go run .

# Подключиться к БД
make db
```

## Переменные окружения

Файл `.env` (скопировать из `.env.example`):

```
TELEGRAM_BOT_TOKEN=   # токен от @BotFather
ALLOWED_CHAT_ID=      # ID группового чата (отрицательное число)
POSTGRES_USER=gymbot
POSTGRES_PASSWORD=
POSTGRES_DB=gymbot
DATABASE_URL=         # автоматически подставляется в docker-compose
```

## База данных

Таблица `polls` создаётся автоматически при старте (`database.go:init`).
Уникальное ограничение: одна запись на `(chat_id, week_number, year)`.

## Архитектурные решения

- Бот работает только с одним чатом (`ALLOWED_CHAT_ID`) — все остальные сообщения игнорируются.
- Номер недели определяется по стандарту ISO 8601 (`time.ISOWeek()`).
- Graceful shutdown через `os.Signal` / `syscall.SIGTERM`.
- Схема БД создаётся через `CREATE TABLE IF NOT EXISTS` при каждом старте — миграций нет.

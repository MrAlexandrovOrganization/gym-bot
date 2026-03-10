package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Poll представляет опрос в базе данных
type Poll struct {
	ID         int
	ChatID     int64
	MessageID  int
	PollID     string
	WeekNumber int
	Year       int
	CreatedAt  time.Time
}

// Database управляет подключением к базе данных
type Database struct {
	db *sql.DB
}

// NewDatabase создаёт новое подключение к базе данных
func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия базы данных: %w", err)
	}

	database := &Database{db: db}
	if err := database.init(); err != nil {
		return nil, err
	}

	return database, nil
}

// init инициализирует таблицы в базе данных
func (d *Database) init() error {
	query := `
	CREATE TABLE IF NOT EXISTS polls (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER NOT NULL,
		message_id INTEGER NOT NULL,
		poll_id TEXT NOT NULL UNIQUE,
		week_number INTEGER NOT NULL,
		year INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(chat_id, week_number, year)
	)
	`

	if _, err := d.db.Exec(query); err != nil {
		return fmt.Errorf("ошибка создания таблицы: %w", err)
	}

	return nil
}

// GetWeekNumber возвращает номер недели для заданной даты
func GetWeekNumber(date time.Time) int {
	_, week := date.ISOWeek()
	return week
}

// HasWeekPoll проверяет, существует ли опрос для текущей недели
func (d *Database) HasWeekPoll(chatID int64) (bool, error) {
	now := time.Now()
	weekNumber := GetWeekNumber(now)
	year := now.Year()

	var count int
	query := `SELECT COUNT(*) FROM polls WHERE chat_id = ? AND week_number = ? AND year = ?`
	err := d.db.QueryRow(query, chatID, weekNumber, year).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки опроса: %w", err)
	}

	return count > 0, nil
}

// SavePoll сохраняет опрос в базу данных
func (d *Database) SavePoll(chatID int64, messageID int, pollID string) error {
	now := time.Now()
	weekNumber := GetWeekNumber(now)
	year := now.Year()

	query := `
	INSERT INTO polls (chat_id, message_id, poll_id, week_number, year)
	VALUES (?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query, chatID, messageID, pollID, weekNumber, year)
	if err != nil {
		return fmt.Errorf("ошибка сохранения опроса: %w", err)
	}

	return nil
}

// GetCurrentWeekPoll возвращает опрос текущей недели
func (d *Database) GetCurrentWeekPoll(chatID int64) (*Poll, error) {
	now := time.Now()
	weekNumber := GetWeekNumber(now)
	year := now.Year()

	query := `
	SELECT id, chat_id, message_id, poll_id, week_number, year, created_at
	FROM polls
	WHERE chat_id = ? AND week_number = ? AND year = ?
	`

	poll := &Poll{}
	err := d.db.QueryRow(query, chatID, weekNumber, year).Scan(
		&poll.ID,
		&poll.ChatID,
		&poll.MessageID,
		&poll.PollID,
		&poll.WeekNumber,
		&poll.Year,
		&poll.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка получения опроса: %w", err)
	}

	return poll, nil
}

// Close закрывает подключение к базе данных
func (d *Database) Close() error {
	return d.db.Close()
}
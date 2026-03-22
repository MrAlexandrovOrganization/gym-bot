package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
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

type DB interface {
	HasWeekPoll(chatID int64, weekNuber, year int) (bool, error)
	SavePoll(chatID int64, messageID int, pollID string, weekNumber, year int) error
	GetWeekPoll(chatID int64, weekNumber, year int) (*Poll, error)
	Close() error
}

// Database управляет подключением к базе данных
type Database struct {
	db *sql.DB
}

// NewDatabase создаёт новое подключение к базе данных PostgreSQL
func NewDatabase(connStr string) (*Database, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия базы данных: %w", err)
	}

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %w", err)
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
		id SERIAL PRIMARY KEY,
		chat_id BIGINT NOT NULL,
		message_id INTEGER NOT NULL,
		poll_id TEXT NOT NULL UNIQUE,
		week_number INTEGER NOT NULL,
		year INTEGER NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(chat_id, week_number, year)
	)
	`

	if _, err := d.db.Exec(query); err != nil {
		return fmt.Errorf("ошибка создания таблицы: %w", err)
	}

	return nil
}

func getWeekNumber(date time.Time) int {
	_, week := date.ISOWeek()
	return week
}

func GetCurrentWeekNumber() int {
	return getWeekNumber(time.Now())
}

func GetYear(date time.Time) int {
	return date.Year()
}

func GetCurrentYear() int {
	return GetYear(time.Now())
}

func getWeekAndYear(date time.Time) (int, int) {
	return getWeekNumber(date), GetYear(date)
}

func getCurrentWeekAndYear() (int, int) {
	return getWeekAndYear(time.Now())
}

func (d *Database) HasWeekPoll(chatID int64, weekNumber int, year int) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM polls WHERE chat_id = $1 AND week_number = $2 AND year = $3`
	err := d.db.QueryRow(query, chatID, weekNumber, year).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки опроса: %w", err)
	}

	return count > 0, nil
}

// HasWeekPoll проверяет, существует ли опрос для текущей недели
func (d *Database) HasCurrentWeekPoll(chatID int64) (bool, error) {
	weekNumber, year := getCurrentWeekAndYear()

	return d.HasWeekPoll(chatID, weekNumber, year)
}

func (d *Database) SavePoll(chatID int64, messageID int, pollID string, weekNumber int, year int) error {
	query := `
	INSERT INTO polls (chat_id, message_id, poll_id, week_number, year)
	VALUES ($1, $2, $3, $4, $5)
	`

	_, err := d.db.Exec(query, chatID, messageID, pollID, weekNumber, year)
	if err != nil {
		return fmt.Errorf("ошибка сохранения опроса: %w", err)
	}

	return nil
}

// SaveCurrentPoll сохраняет опрос в базу данных
func (d *Database) SaveCurrentPoll(chatID int64, messageID int, pollID string) error {
	weekNumber, year := getCurrentWeekAndYear()

	return d.SavePoll(chatID, messageID, pollID, weekNumber, year)
}

func (d *Database) GetWeekPoll(chatID int64, weekNumber int, year int) (*Poll, error) {
	query := `
	SELECT id, chat_id, message_id, poll_id, week_number, year, created_at
	FROM polls
	WHERE chat_id = $1 AND week_number = $2 AND year = $3
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

// GetCurrentWeekPoll возвращает опрос текущей недели
func (d *Database) GetCurrentWeekPoll(chatID int64) (*Poll, error) {
	weekNumber, year := getCurrentWeekAndYear()

	return d.GetWeekPoll(chatID, weekNumber, year)
}

// Close закрывает подключение к базе данных
func (d *Database) Close() error {
	return d.db.Close()
}

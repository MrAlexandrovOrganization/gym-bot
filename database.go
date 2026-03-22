package main

import (
	"database/sql"
	"errors"
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
	HasWeekPoll(chatID int64, weekNumber, year int) (bool, error)
	SavePoll(chatID int64, messageID int, pollID string, weekNumber, year int) error
	GetWeekPoll(chatID int64, weekNumber, year int) (*Poll, error)
	Close() error
}

// Database управляет подключением к базе данных
type Database struct {
	db *sql.DB
}

func NewDatabase(databaseURL string) (*Database, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is empty")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия базы данных: %w", err)
	}

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

func (d *Database) HasWeekPoll(chatID int64, weekNumber int, year int) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM polls WHERE chat_id = $1 AND week_number = $2 AND year = $3`
	err := d.db.QueryRow(query, chatID, weekNumber, year).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки опроса: %w", err)
	}

	return count > 0, nil
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

	if errors.Is(err, sql.ErrNoRows) {
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

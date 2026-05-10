package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

type Deadline struct {
	ID          int64
	UserID      int64
	Text        string
	DeadlineAt  time.Time
	Reminded24h bool
	Reminded3h  bool
	CreatedAt   time.Time
}

func InitDB(dsn string) (*DB, error) {
	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("cannot connect to postgres: %w", err)
	}
	_, err = sqlDB.Exec(`
		CREATE TABLE IF NOT EXISTS deadlines (
			id           BIGSERIAL PRIMARY KEY,
			user_id      BIGINT       NOT NULL,
			text         TEXT         NOT NULL,
			deadline_at  TIMESTAMPTZ  NOT NULL,
			reminded_24h BOOLEAN      DEFAULT FALSE,
			reminded_12h BOOLEAN      DEFAULT FALSE,
			created_at   TIMESTAMPTZ  DEFAULT NOW()
		)
	`)
	if err != nil {
		return nil, err
	}
	return &DB{sqlDB}, nil
}

func (db *DB) AddDeadline(userID int64, text string, deadlineAt time.Time) error {
	_, err := db.Exec(
		`INSERT INTO deadlines (user_id, text, deadline_at) VALUES ($1, $2, $3)`,
		userID, text, deadlineAt,
	)
	return err
}

func (db *DB) ListDeadlines(userID int64) ([]Deadline, error) {
	rows, err := db.Query(`
		SELECT id, text, deadline_at, created_at
		FROM deadlines
		WHERE user_id = $1 AND deadline_at > $2
		ORDER BY deadline_at ASC
	`, userID, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Deadline
	for rows.Next() {
		var d Deadline
		d.UserID = userID
		if err := rows.Scan(&d.ID, &d.Text, &d.DeadlineAt, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (db *DB) DeleteAllDeadlines(userID int64) (int64, error) {
	res, err := db.Exec(`DELETE FROM deadlines WHERE user_id = $1`, userID)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (db *DB) UpdateDeadline(id, userID int64, text string, deadlineAt time.Time) error {
	res, err := db.Exec(`
		UPDATE deadlines
		SET text = $1, deadline_at = $2, reminded_24h = FALSE, reminded_12h = FALSE
		WHERE id = $3 AND user_id = $4
	`, text, deadlineAt, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("deadline #%d not found", id)
	}
	return nil
}

func (db *DB) DeleteByIDs(userID int64, ids []int64) (int64, error) {
	res, err := db.Exec(
		`DELETE FROM deadlines WHERE user_id = $1 AND id = ANY($2)`,
		userID, pq.Array(ids),
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (db *DB) GetDeadlineByID(id, userID int64) (*Deadline, error) {
	var d Deadline
	d.UserID = userID
	err := db.QueryRow(`
		SELECT id, text, deadline_at, created_at FROM deadlines WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&d.ID, &d.Text, &d.DeadlineAt, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("deadline #%d not found", id)
	}
	return &d, err
}

func (db *DB) GetPendingReminders(targetDuration time.Duration, flag string) ([]Deadline, error) {
	now := time.Now()
	from := now.Add(targetDuration - time.Hour)
	to := now.Add(targetDuration + time.Hour)

	col := reminderColumn(flag)
	if col == "" {
		return nil, fmt.Errorf("unknown reminder flag: %s", flag)
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, text, deadline_at
		FROM deadlines
		WHERE deadline_at BETWEEN $1 AND $2 AND %s = FALSE
	`, col)

	rows, err := db.Query(query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Deadline
	for rows.Next() {
		var d Deadline
		if err := rows.Scan(&d.ID, &d.UserID, &d.Text, &d.DeadlineAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (db *DB) MarkReminded(id int64, flag string) error {
	col := reminderColumn(flag)
	if col == "" {
		return fmt.Errorf("unknown reminder flag: %s", flag)
	}
	_, err := db.Exec(fmt.Sprintf(`UPDATE deadlines SET %s = TRUE WHERE id = $1`, col), id)
	return err
}

func reminderColumn(flag string) string {
	switch flag {
	case "24h":
		return "reminded_24h"
	case "12h":
		return "reminded_12h"
	}
	return ""
}

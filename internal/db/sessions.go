package db

import (
	"context"
	"database/sql"
)

// Session represents a row in the sessions table.
type Session struct {
	ID        string  `json:"id"`
	UserID    int64   `json:"user_id"`
	Title     *string `json:"title,omitempty"`
	Model     string  `json:"model"`
	CreatedAt string  `json:"created_at"`
}

// CreateSession inserts a new session.
func CreateSession(ctx context.Context, db *DB, id string, userID int64, title *string, model string) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, title, model) VALUES (?, ?, ?, ?)`,
		id, userID, title, model,
	)
	return err
}

// ListSessions returns all sessions for a user, newest first.
func ListSessions(ctx context.Context, db *DB, userID int64) ([]Session, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, user_id, title, model, created_at FROM sessions WHERE user_id = ? ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.UserID, &s.Title, &s.Model, &s.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// GetSession looks up a session by ID.
func GetSession(ctx context.Context, db *DB, id string) (*Session, error) {
	s := &Session{}
	err := db.QueryRowContext(ctx,
		`SELECT id, user_id, title, model, created_at FROM sessions WHERE id = ?`,
		id,
	).Scan(&s.ID, &s.UserID, &s.Title, &s.Model, &s.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

// UpdateSessionTitle sets the title for a session.
func UpdateSessionTitle(ctx context.Context, db *DB, id string, title *string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE sessions SET title = ? WHERE id = ?`,
		title, id,
	)
	return err
}

// DeleteSession deletes a session owned by the given user. Returns sql.ErrNoRows
// if the session doesn't exist or doesn't belong to the user.
func DeleteSession(ctx context.Context, db *DB, id string, userID int64) error {
	res, err := db.ExecContext(ctx,
		`DELETE FROM sessions WHERE id = ? AND user_id = ?`,
		id, userID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

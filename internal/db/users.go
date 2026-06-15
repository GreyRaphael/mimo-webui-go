package db

import (
	"context"
	"database/sql"
)

// User represents a row in the users table.
type User struct {
	ID           int64   `json:"id"`
	Username     string  `json:"username"`
	PasswordHash string  `json:"-"`
	Role         string  `json:"role"`
	CreatedAt    string  `json:"created_at"`
	LastLogin    *string `json:"last_login,omitempty"`
}

// CreateUser inserts a new user and returns its ID.
func CreateUser(ctx context.Context, db *DB, username, passwordHash, role string) (int64, error) {
	res, err := db.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`,
		username, passwordHash, role,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetUserByUsername looks up a user by username.
func GetUserByUsername(ctx context.Context, db *DB, username string) (*User, error) {
	u := &User{}
	err := db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, role, created_at, last_login FROM users WHERE username = ?`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.LastLogin)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

// GetUserByID looks up a user by ID.
func GetUserByID(ctx context.Context, db *DB, id int64) (*User, error) {
	u := &User{}
	err := db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, role, created_at, last_login FROM users WHERE id = ?`,
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.LastLogin)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

// UpdateLastLogin sets last_login to the current time for the given user.
func UpdateLastLogin(ctx context.Context, db *DB, id int64) error {
	_, err := db.ExecContext(ctx,
		`UPDATE users SET last_login = datetime('now') WHERE id = ?`,
		id,
	)
	return err
}

// CountUsers returns the total number of users.
func CountUsers(ctx context.Context, db *DB) (int, error) {
	var n int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

// ListUsers returns all users ordered by creation time.
func ListUsers(ctx context.Context, db *DB) ([]User, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, username, password_hash, role, created_at, last_login FROM users ORDER BY created_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.LastLogin); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

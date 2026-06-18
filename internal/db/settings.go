package db

import "context"

// GetSetting returns a setting value for a user, or empty string if not set.
func GetSetting(ctx context.Context, db *DB, userID int64, name string) (string, error) {
	var value string
	err := db.QueryRowContext(ctx,
		`SELECT value FROM settings WHERE user_id = ? AND name = ?`,
		userID, name,
	).Scan(&value)
	if err != nil {
		return "", nil // not found is not an error
	}
	return value, nil
}

// SetSetting upserts a setting value for a user.
func SetSetting(ctx context.Context, db *DB, userID int64, name, value string) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO settings (user_id, name, value) VALUES (?, ?, ?)
		 ON CONFLICT(user_id, name) DO UPDATE SET value = excluded.value`,
		userID, name, value,
	)
	return err
}

// GetSettings returns all settings for a user as a map.
func GetSettings(ctx context.Context, db *DB, userID int64) (map[string]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT name, value FROM settings WHERE user_id = ?`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		result[name] = value
	}
	return result, rows.Err()
}

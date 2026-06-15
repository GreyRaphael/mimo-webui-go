package db

import "context"

// Message represents a row in the messages table.
type Message struct {
	ID        int64   `json:"id"`
	SessionID string  `json:"session_id"`
	Role      string  `json:"role"`
	Content   *string `json:"content,omitempty"`
	MediaType *string `json:"media_type,omitempty"`
	MediaURL  *string `json:"media_url,omitempty"`
	Reasoning *string `json:"reasoning,omitempty"`
	CreatedAt string  `json:"created_at"`
}

// CreateMessage inserts a new message and returns its ID.
func CreateMessage(ctx context.Context, db *DB, sessionID, role string, content, mediaType, mediaURL, reasoning *string) (int64, error) {
	res, err := db.ExecContext(ctx,
		`INSERT INTO messages (session_id, role, content, media_type, media_url, reasoning) VALUES (?, ?, ?, ?, ?, ?)`,
		sessionID, role, content, mediaType, mediaURL, reasoning,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListMessages returns messages for a session, oldest first, limited by limit.
func ListMessages(ctx context.Context, db *DB, sessionID string, limit int) ([]Message, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, session_id, role, content, media_type, media_url, reasoning, created_at
		 FROM messages WHERE session_id = ? ORDER BY created_at ASC LIMIT ?`,
		sessionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	msgs := make([]Message, 0)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.MediaType, &m.MediaURL, &m.Reasoning, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

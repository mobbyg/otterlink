package db

import (
	"database/sql"
	"fmt"
)

const schema = `
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE COLLATE NOCASE,
    display_name TEXT NOT NULL,
    email TEXT UNIQUE COLLATE NOCASE,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    role TEXT NOT NULL DEFAULT 'user',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TEXT NOT NULL,
    last_seen TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

CREATE TABLE IF NOT EXISTS buddies (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    buddy_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, buddy_id),
    CHECK (user_id <> buddy_id)
);

CREATE INDEX IF NOT EXISTS idx_buddies_buddy_id ON buddies(buddy_id);
`

func Initialize(db *sql.DB) error {
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("initialize schema: %w", err)
	}

	// Existing databases predate roles. SQLite does not provide
	// ALTER TABLE ... ADD COLUMN IF NOT EXISTS, so inspect the table first.
	rows, err := db.Query(`PRAGMA table_info(users)`)
	if err != nil {
		return fmt.Errorf("inspect users schema: %w", err)
	}
	defer rows.Close()

	hasRole := false
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, pk int
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("read users schema: %w", err)
		}
		if name == "role" {
			hasRole = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read users schema rows: %w", err)
	}
	if !hasRole {
		if _, err := db.Exec(`ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'user'`); err != nil {
			return fmt.Errorf("add users role column: %w", err)
		}
	}
	return nil
}

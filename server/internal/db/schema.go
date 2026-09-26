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
    PRIMARY KEY (user_id, buddy_id), CHECK (user_id <> buddy_id)
);
CREATE INDEX IF NOT EXISTS idx_buddies_buddy_id ON buddies(buddy_id);

CREATE TABLE IF NOT EXISTS audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    actor_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    actor_username TEXT NOT NULL,
    action TEXT NOT NULL,
    target_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    target_username TEXT,
    result TEXT NOT NULL,
    details TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON audit_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_actor_user_id ON audit_log(actor_user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_target_user_id ON audit_log(target_user_id);

CREATE TABLE IF NOT EXISTS chat_channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE,
    creator_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    creator_username TEXT NOT NULL,
    original_mod_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    original_mod_username TEXT,
    permanent INTEGER NOT NULL DEFAULT 0,
    allow_ops_to_create_ops INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS chat_channel_members (
    channel_id INTEGER NOT NULL REFERENCES chat_channels(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'user',
    joined_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (channel_id, user_id), CHECK (role IN ('user', 'op', 'mod', 'original_mod'))
);
CREATE INDEX IF NOT EXISTS idx_chat_channel_members_user_id ON chat_channel_members(user_id);

CREATE TABLE IF NOT EXISTS chat_channel_bans (
    channel_id INTEGER NOT NULL REFERENCES chat_channels(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (channel_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_chat_channel_bans_user_id ON chat_channel_bans(user_id);

CREATE TABLE IF NOT EXISTS chat_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id INTEGER NOT NULL REFERENCES chat_channels(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_chat_messages_channel_id ON chat_messages(channel_id, id);

CREATE TABLE IF NOT EXISTS direct_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sender_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    read_at TEXT,
    CHECK (sender_id <> recipient_id)
);
CREATE INDEX IF NOT EXISTS idx_direct_messages_pair ON direct_messages(sender_id, recipient_id, id);
CREATE INDEX IF NOT EXISTS idx_direct_messages_recipient_unread ON direct_messages(recipient_id, read_at, id);

CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    event_type TEXT NOT NULL CHECK (event_type IN ('public', 'community', 'server')),
    created_by TEXT NOT NULL,
    target_type TEXT NOT NULL DEFAULT 'none' CHECK (target_type IN ('none', 'chat')),
    target_id INTEGER,
    start_at TEXT NOT NULL,
    end_at TEXT NOT NULL,
    all_day INTEGER NOT NULL DEFAULT 0 CHECK (all_day IN (0,1)),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_events_start_at ON events(start_at);
CREATE INDEX IF NOT EXISTS idx_events_target ON events(target_type, target_id);\n\nCREATE TABLE IF NOT EXISTS news_sources (\n    id INTEGER PRIMARY KEY AUTOINCREMENT,\n    name TEXT NOT NULL,\n    url TEXT NOT NULL UNIQUE,\n    category TEXT NOT NULL,\n    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0,1)),\n    refresh_interval_minutes INTEGER NOT NULL DEFAULT 30,\n    last_fetched_at TEXT,\n    last_success_at TEXT,\n    last_error TEXT NOT NULL DEFAULT '',\n    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,\n    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP\n);\nCREATE INDEX IF NOT EXISTS idx_news_sources_enabled ON news_sources(enabled);
`

func Initialize(db *sql.DB) error {
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil { return fmt.Errorf("enable foreign keys: %w", err) }
	if _, err := db.Exec(schema); err != nil { return fmt.Errorf("initialize database schema: %w", err) }

	rows, err := db.Query(`PRAGMA table_info(users)`)
	if err != nil { return fmt.Errorf("inspect users schema: %w", err) }
	defer rows.Close()
	hasRole := false
	for rows.Next() {
		var cid int; var name, columnType string; var notNull, pk int; var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil { return fmt.Errorf("read users schema: %w", err) }
		if name == "role" { hasRole = true }
	}
	if err := rows.Err(); err != nil { return fmt.Errorf("read users schema rows: %w", err) }
	if !hasRole { if _, err := db.Exec(`ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'user'`); err != nil { return fmt.Errorf("add users role column: %w", err) } }

	if err := addColumnIfMissing(db, "chat_channels", "creator_username", `ALTER TABLE chat_channels ADD COLUMN creator_username TEXT NOT NULL DEFAULT ''`); err != nil { return err }
	if err := addColumnIfMissing(db, "chat_channels", "original_mod_username", `ALTER TABLE chat_channels ADD COLUMN original_mod_username TEXT`); err != nil { return err }
	if _, err := db.Exec(`UPDATE chat_channels SET creator_username = COALESCE((SELECT username FROM users WHERE users.id = chat_channels.creator_user_id), '') WHERE creator_username = ''`); err != nil { return fmt.Errorf("backfill chat creator usernames: %w", err) }
	if _, err := db.Exec(`UPDATE chat_channels SET original_mod_username = (SELECT username FROM users WHERE users.id = chat_channels.original_mod_user_id) WHERE original_mod_username IS NULL AND original_mod_user_id IS NOT NULL`); err != nil { return fmt.Errorf("backfill original mod usernames: %w", err) }
	return nil
}

func addColumnIfMissing(db *sql.DB, table, column, alter string) error {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil { return fmt.Errorf("inspect %s schema: %w", table, err) }
	defer rows.Close()
	for rows.Next() {
		var cid int; var name, columnType string; var notNull, pk int; var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil { return fmt.Errorf("read %s schema: %w", table, err) }
		if name == column { return nil }
	}
	if err := rows.Err(); err != nil { return fmt.Errorf("read %s schema rows: %w", table, err) }
	if _, err := db.Exec(alter); err != nil { return fmt.Errorf("add %s.%s: %w", table, column, err) }
	return nil
}

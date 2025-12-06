-- Migration 001: Initialize core schema
-- Created: 2025-12-06
-- Purpose: Create channels, users, and messages tables with FTS5 support

-- Channels table
CREATE TABLE IF NOT EXISTS channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE,
    display_name TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    retain_history_on_delete INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    last_message_at TEXT,
    total_messages INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_channels_enabled ON channels(enabled);
CREATE INDEX IF NOT EXISTS idx_channels_name ON channels(name);

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE COLLATE NOCASE,
    display_name TEXT,
    first_seen_at TEXT NOT NULL DEFAULT (datetime('now')),
    last_seen_at TEXT NOT NULL DEFAULT (datetime('now')),
    total_messages INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_last_seen ON users(last_seen_at);

-- Messages table
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    text TEXT NOT NULL,
    sent_at TEXT NOT NULL DEFAULT (datetime('now')),
    tags TEXT,
    FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_messages_channel_id ON messages(channel_id);
CREATE INDEX IF NOT EXISTS idx_messages_user_id ON messages(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_sent_at ON messages(sent_at);
CREATE INDEX IF NOT EXISTS idx_messages_channel_sent ON messages(channel_id, sent_at);

-- FTS5 virtual table for message search (optional, can be disabled)
CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
    content,
    message_id UNINDEXED,
    content='messages',
    content_rowid='id'
);

-- Triggers to keep FTS in sync with messages table
CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
    INSERT INTO messages_fts(rowid, content, message_id) VALUES (new.id, new.text, new.id);
END;

CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, content, message_id) VALUES('delete', old.id, old.text, old.id);
END;

CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, content, message_id) VALUES('delete', old.id, old.text, old.id);
    INSERT INTO messages_fts(rowid, content, message_id) VALUES (new.id, new.text, new.id);
END;

-- Trigger to update channel stats on message insert
CREATE TRIGGER IF NOT EXISTS update_channel_stats AFTER INSERT ON messages BEGIN
    UPDATE channels 
    SET total_messages = total_messages + 1,
        last_message_at = new.sent_at,
        updated_at = datetime('now')
    WHERE id = new.channel_id;
END;

-- Trigger to update user stats on message insert
CREATE TRIGGER IF NOT EXISTS update_user_stats AFTER INSERT ON messages BEGIN
    UPDATE users 
    SET total_messages = total_messages + 1,
        last_seen_at = new.sent_at
    WHERE id = new.user_id;
END;

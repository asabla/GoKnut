// Package repository provides database access for the Twitch Chat Archiver.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Message represents a message in the database.
type Message struct {
	ID          int64
	ChannelID   int64
	UserID      int64
	Text        string
	SentAt      time.Time
	Tags        map[string]string
	Username    string // Joined from users table
	DisplayName string // Joined from users table
	ChannelName string // Joined from channels table
}

// User represents a user in the database.
type User struct {
	ID            int64
	Username      string
	DisplayName   string
	FirstSeenAt   time.Time
	LastSeenAt    time.Time
	TotalMessages int64
}

// MessageRepository provides operations for messages.
type MessageRepository struct {
	db *DB
}

// NewMessageRepository creates a new message repository.
func NewMessageRepository(db *DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// Create inserts a new message into the database.
func (r *MessageRepository) Create(ctx context.Context, msg *Message) error {
	var tagsJSON sql.NullString
	if len(msg.Tags) > 0 {
		data, err := json.Marshal(msg.Tags)
		if err != nil {
			return fmt.Errorf("failed to marshal tags: %w", err)
		}
		tagsJSON = sql.NullString{String: string(data), Valid: true}
	}

	query := `
		INSERT INTO messages (channel_id, user_id, text, sent_at, tags)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		msg.ChannelID, msg.UserID, msg.Text, msg.SentAt.Format(time.RFC3339), tagsJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	msg.ID = id

	return nil
}

// CreateBatch inserts multiple messages in a single transaction.
func (r *MessageRepository) CreateBatch(ctx context.Context, messages []Message) error {
	if len(messages) == 0 {
		return nil
	}

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO messages (channel_id, user_id, text, sent_at, tags)
			VALUES (?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("failed to prepare statement: %w", err)
		}
		defer stmt.Close()

		for i := range messages {
			msg := &messages[i]

			var tagsJSON sql.NullString
			if len(msg.Tags) > 0 {
				data, err := json.Marshal(msg.Tags)
				if err != nil {
					return fmt.Errorf("failed to marshal tags: %w", err)
				}
				tagsJSON = sql.NullString{String: string(data), Valid: true}
			}

			result, err := stmt.ExecContext(ctx,
				msg.ChannelID, msg.UserID, msg.Text, msg.SentAt.Format(time.RFC3339), tagsJSON,
			)
			if err != nil {
				return fmt.Errorf("failed to insert message: %w", err)
			}

			id, err := result.LastInsertId()
			if err != nil {
				return fmt.Errorf("failed to get last insert id: %w", err)
			}
			msg.ID = id
		}

		return nil
	})
}

// GetByID returns a message by ID.
func (r *MessageRepository) GetByID(ctx context.Context, id int64) (*Message, error) {
	query := `
		SELECT m.id, m.channel_id, m.user_id, m.text, m.sent_at, m.tags,
		       u.username, u.display_name, c.name as channel_name
		FROM messages m
		JOIN users u ON m.user_id = u.id
		JOIN channels c ON m.channel_id = c.id
		WHERE m.id = ?
	`

	return r.scanMessage(r.db.QueryRowContext(ctx, query, id))
}

// GetRecent returns the most recent messages for a channel.
func (r *MessageRepository) GetRecent(ctx context.Context, channelID int64, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		SELECT m.id, m.channel_id, m.user_id, m.text, m.sent_at, m.tags,
		       u.username, u.display_name, c.name as channel_name
		FROM messages m
		JOIN users u ON m.user_id = u.id
		JOIN channels c ON m.channel_id = c.id
		WHERE m.channel_id = ?
		ORDER BY m.sent_at DESC, m.id DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, channelID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent messages: %w", err)
	}
	defer rows.Close()

	return r.scanMessages(rows)
}

// GetPaginated returns paginated messages for a channel.
func (r *MessageRepository) GetPaginated(ctx context.Context, channelID int64, page, pageSize int) ([]Message, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	// Get total count
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM messages WHERE channel_id = ?`
	if err := r.db.QueryRowContext(ctx, countQuery, channelID).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count messages: %w", err)
	}

	// Get paginated results
	query := `
		SELECT m.id, m.channel_id, m.user_id, m.text, m.sent_at, m.tags,
		       u.username, u.display_name, c.name as channel_name
		FROM messages m
		JOIN users u ON m.user_id = u.id
		JOIN channels c ON m.channel_id = c.id
		WHERE m.channel_id = ?
		ORDER BY m.sent_at DESC, m.id DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, channelID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query paginated messages: %w", err)
	}
	defer rows.Close()

	messages, err := r.scanMessages(rows)
	if err != nil {
		return nil, 0, err
	}

	return messages, totalCount, nil
}

// GetBeforeID returns messages before the given ID for cursor-based pagination.
func (r *MessageRepository) GetBeforeID(ctx context.Context, channelID, beforeID int64, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		SELECT m.id, m.channel_id, m.user_id, m.text, m.sent_at, m.tags,
		       u.username, u.display_name, c.name as channel_name
		FROM messages m
		JOIN users u ON m.user_id = u.id
		JOIN channels c ON m.channel_id = c.id
		WHERE m.channel_id = ? AND m.id < ?
		ORDER BY m.sent_at DESC, m.id DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, channelID, beforeID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages before ID: %w", err)
	}
	defer rows.Close()

	return r.scanMessages(rows)
}

// GetAfterID returns messages after the given ID for streaming/polling.
func (r *MessageRepository) GetAfterID(ctx context.Context, channelID, afterID int64, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		SELECT m.id, m.channel_id, m.user_id, m.text, m.sent_at, m.tags,
		       u.username, u.display_name, c.name as channel_name
		FROM messages m
		JOIN users u ON m.user_id = u.id
		JOIN channels c ON m.channel_id = c.id
		WHERE m.channel_id = ? AND m.id > ?
		ORDER BY m.sent_at ASC, m.id ASC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, channelID, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages after ID: %w", err)
	}
	defer rows.Close()

	return r.scanMessages(rows)
}

// GetLatestID returns the ID of the most recent message for a channel.
func (r *MessageRepository) GetLatestID(ctx context.Context, channelID int64) (int64, error) {
	query := `SELECT COALESCE(MAX(id), 0) FROM messages WHERE channel_id = ?`

	var latestID int64
	if err := r.db.QueryRowContext(ctx, query, channelID).Scan(&latestID); err != nil {
		return 0, fmt.Errorf("failed to get latest message ID: %w", err)
	}

	return latestID, nil
}

// GetRecentGlobal returns the most recent messages across all channels.
func (r *MessageRepository) GetRecentGlobal(ctx context.Context, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		SELECT m.id, m.channel_id, m.user_id, m.text, m.sent_at, m.tags,
		       u.username, u.display_name, c.name as channel_name
		FROM messages m
		JOIN users u ON m.user_id = u.id
		JOIN channels c ON m.channel_id = c.id
		ORDER BY m.sent_at DESC, m.id DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent global messages: %w", err)
	}
	defer rows.Close()

	return r.scanMessages(rows)
}

// GetTotalCount returns the total number of messages in the database.
func (r *MessageRepository) GetTotalCount(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM messages`

	var count int64
	if err := r.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return count, nil
}

// GetGlobalAfterID returns messages after the given ID across all channels for backfill.
func (r *MessageRepository) GetGlobalAfterID(ctx context.Context, afterID int64, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	query := `
		SELECT m.id, m.channel_id, m.user_id, m.text, m.sent_at, m.tags,
		       u.username, u.display_name, c.name as channel_name
		FROM messages m
		JOIN users u ON m.user_id = u.id
		JOIN channels c ON m.channel_id = c.id
		WHERE m.id > ?
		ORDER BY m.id ASC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query global messages after ID: %w", err)
	}
	defer rows.Close()

	return r.scanMessages(rows)
}

// GetGlobalLatestID returns the ID of the most recent message across all channels.
func (r *MessageRepository) GetGlobalLatestID(ctx context.Context) (int64, error) {
	query := `SELECT COALESCE(MAX(id), 0) FROM messages`

	var latestID int64
	if err := r.db.QueryRowContext(ctx, query).Scan(&latestID); err != nil {
		return 0, fmt.Errorf("failed to get global latest message ID: %w", err)
	}

	return latestID, nil
}

func (r *MessageRepository) scanMessage(row *sql.Row) (*Message, error) {
	var msg Message
	var sentAt string
	var tagsJSON sql.NullString

	err := row.Scan(
		&msg.ID, &msg.ChannelID, &msg.UserID, &msg.Text, &sentAt, &tagsJSON,
		&msg.Username, &msg.DisplayName, &msg.ChannelName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan message: %w", err)
	}

	msg.SentAt, _ = ParseSQLiteDatetime(sentAt)

	if tagsJSON.Valid && tagsJSON.String != "" {
		if err := json.Unmarshal([]byte(tagsJSON.String), &msg.Tags); err != nil {
			// Non-fatal, just ignore tags
			msg.Tags = nil
		}
	}

	return &msg, nil
}

func (r *MessageRepository) scanMessages(rows *sql.Rows) ([]Message, error) {
	var messages []Message

	for rows.Next() {
		var msg Message
		var sentAt string
		var tagsJSON sql.NullString

		err := rows.Scan(
			&msg.ID, &msg.ChannelID, &msg.UserID, &msg.Text, &sentAt, &tagsJSON,
			&msg.Username, &msg.DisplayName, &msg.ChannelName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		msg.SentAt, _ = ParseSQLiteDatetime(sentAt)

		if tagsJSON.Valid && tagsJSON.String != "" {
			if err := json.Unmarshal([]byte(tagsJSON.String), &msg.Tags); err != nil {
				// Non-fatal, just ignore tags
				msg.Tags = nil
			}
		}

		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// UserRepository provides operations for users.
type UserRepository struct {
	db *DB
}

// NewUserRepository creates a new user repository.
func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetByID returns a user by ID.
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*User, error) {
	query := `
		SELECT id, username, display_name, first_seen_at, last_seen_at, total_messages
		FROM users
		WHERE id = ?
	`

	var user User
	var firstSeen, lastSeen string
	var displayName sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Username, &displayName, &firstSeen, &lastSeen, &user.TotalMessages,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.FirstSeenAt, _ = ParseSQLiteDatetime(firstSeen)
	user.LastSeenAt, _ = ParseSQLiteDatetime(lastSeen)
	if displayName.Valid {
		user.DisplayName = displayName.String
	}

	return &user, nil
}

// GetByUsername returns a user by username.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	query := `
		SELECT id, username, display_name, first_seen_at, last_seen_at, total_messages
		FROM users
		WHERE username = ?
	`

	var user User
	var firstSeen, lastSeen string
	var displayName sql.NullString

	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID, &user.Username, &displayName, &firstSeen, &lastSeen, &user.TotalMessages,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	user.FirstSeenAt, _ = ParseSQLiteDatetime(firstSeen)
	user.LastSeenAt, _ = ParseSQLiteDatetime(lastSeen)
	if displayName.Valid {
		user.DisplayName = displayName.String
	}

	return &user, nil
}

// Create creates a new user.
func (r *UserRepository) Create(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (username, display_name, first_seen_at, last_seen_at)
		VALUES (?, ?, datetime('now'), datetime('now'))
	`

	var displayName sql.NullString
	if user.DisplayName != "" {
		displayName = sql.NullString{String: user.DisplayName, Valid: true}
	}

	result, err := r.db.ExecContext(ctx, query, user.Username, displayName)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	user.ID = id

	return nil
}

// GetOrCreate returns an existing user or creates a new one.
func (r *UserRepository) GetOrCreate(ctx context.Context, username, displayName string) (*User, error) {
	// Try to get existing user
	user, err := r.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user != nil {
		// Update display name if changed
		if displayName != "" && displayName != user.DisplayName {
			if _, err := r.db.ExecContext(ctx,
				"UPDATE users SET display_name = ? WHERE id = ?",
				displayName, user.ID,
			); err != nil {
				// Non-fatal
			}
			user.DisplayName = displayName
		}
		return user, nil
	}

	// Create new user
	user = &User{
		Username:    username,
		DisplayName: displayName,
	}
	if err := r.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetCount returns the total number of users in the database.
func (r *UserRepository) GetCount(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM users`

	var count int64
	if err := r.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}

// Package repository provides database access for the Twitch Chat Archiver.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Channel represents a channel in the database.
type Channel struct {
	ID                    int64
	Name                  string
	DisplayName           string
	Enabled               bool
	RetainHistoryOnDelete bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
	LastMessageAt         *time.Time
	TotalMessages         int64
}

// ChannelRepository provides CRUD operations for channels.
type ChannelRepository struct {
	db *DB
}

// NewChannelRepository creates a new channel repository.
func NewChannelRepository(db *DB) *ChannelRepository {
	return &ChannelRepository{db: db}
}

// List returns all channels with optional filtering.
func (r *ChannelRepository) List(ctx context.Context) ([]Channel, error) {
	query := `
		SELECT id, name, display_name, enabled, retain_history_on_delete,
		       created_at, updated_at, last_message_at, total_messages
		FROM channels
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query channels: %w", err)
	}
	defer rows.Close()

	var channels []Channel
	for rows.Next() {
		var ch Channel
		var createdAt, updatedAt string
		var lastMessageAt sql.NullString

		err := rows.Scan(
			&ch.ID, &ch.Name, &ch.DisplayName, &ch.Enabled, &ch.RetainHistoryOnDelete,
			&createdAt, &updatedAt, &lastMessageAt, &ch.TotalMessages,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}

		ch.CreatedAt, _ = ParseSQLiteDatetime(createdAt)
		ch.UpdatedAt, _ = ParseSQLiteDatetime(updatedAt)
		if lastMessageAt.Valid {
			t, _ := ParseSQLiteDatetime(lastMessageAt.String)
			ch.LastMessageAt = &t
		}

		channels = append(channels, ch)
	}

	return channels, rows.Err()
}

// ListEnabled returns only enabled channels.
func (r *ChannelRepository) ListEnabled(ctx context.Context) ([]Channel, error) {
	query := `
		SELECT id, name, display_name, enabled, retain_history_on_delete,
		       created_at, updated_at, last_message_at, total_messages
		FROM channels
		WHERE enabled = 1
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query enabled channels: %w", err)
	}
	defer rows.Close()

	var channels []Channel
	for rows.Next() {
		var ch Channel
		var createdAt, updatedAt string
		var lastMessageAt sql.NullString

		err := rows.Scan(
			&ch.ID, &ch.Name, &ch.DisplayName, &ch.Enabled, &ch.RetainHistoryOnDelete,
			&createdAt, &updatedAt, &lastMessageAt, &ch.TotalMessages,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}

		ch.CreatedAt, _ = ParseSQLiteDatetime(createdAt)
		ch.UpdatedAt, _ = ParseSQLiteDatetime(updatedAt)
		if lastMessageAt.Valid {
			t, _ := ParseSQLiteDatetime(lastMessageAt.String)
			ch.LastMessageAt = &t
		}

		channels = append(channels, ch)
	}

	return channels, rows.Err()
}

// GetByID returns a channel by ID.
func (r *ChannelRepository) GetByID(ctx context.Context, id int64) (*Channel, error) {
	query := `
		SELECT id, name, display_name, enabled, retain_history_on_delete,
		       created_at, updated_at, last_message_at, total_messages
		FROM channels
		WHERE id = ?
	`

	var ch Channel
	var createdAt, updatedAt string
	var lastMessageAt sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&ch.ID, &ch.Name, &ch.DisplayName, &ch.Enabled, &ch.RetainHistoryOnDelete,
		&createdAt, &updatedAt, &lastMessageAt, &ch.TotalMessages,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	ch.CreatedAt, _ = ParseSQLiteDatetime(createdAt)
	ch.UpdatedAt, _ = ParseSQLiteDatetime(updatedAt)
	if lastMessageAt.Valid {
		t, _ := ParseSQLiteDatetime(lastMessageAt.String)
		ch.LastMessageAt = &t
	}

	return &ch, nil
}

// GetByName returns a channel by name.
func (r *ChannelRepository) GetByName(ctx context.Context, name string) (*Channel, error) {
	query := `
		SELECT id, name, display_name, enabled, retain_history_on_delete,
		       created_at, updated_at, last_message_at, total_messages
		FROM channels
		WHERE name = ?
	`

	var ch Channel
	var createdAt, updatedAt string
	var lastMessageAt sql.NullString

	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&ch.ID, &ch.Name, &ch.DisplayName, &ch.Enabled, &ch.RetainHistoryOnDelete,
		&createdAt, &updatedAt, &lastMessageAt, &ch.TotalMessages,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get channel by name: %w", err)
	}

	ch.CreatedAt, _ = ParseSQLiteDatetime(createdAt)
	ch.UpdatedAt, _ = ParseSQLiteDatetime(updatedAt)
	if lastMessageAt.Valid {
		t, _ := ParseSQLiteDatetime(lastMessageAt.String)
		ch.LastMessageAt = &t
	}

	return &ch, nil
}

// Create creates a new channel.
func (r *ChannelRepository) Create(ctx context.Context, ch *Channel) error {
	query := `
		INSERT INTO channels (name, display_name, enabled, retain_history_on_delete)
		VALUES (?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		ch.Name, ch.DisplayName, ch.Enabled, ch.RetainHistoryOnDelete,
	)
	if err != nil {
		return fmt.Errorf("failed to create channel: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	ch.ID = id

	// Fetch the created timestamps
	created, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if created != nil {
		ch.CreatedAt = created.CreatedAt
		ch.UpdatedAt = created.UpdatedAt
	}

	return nil
}

// Update updates an existing channel.
func (r *ChannelRepository) Update(ctx context.Context, ch *Channel) error {
	query := `
		UPDATE channels
		SET display_name = ?,
		    enabled = ?,
		    retain_history_on_delete = ?,
		    updated_at = datetime('now')
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		ch.DisplayName, ch.Enabled, ch.RetainHistoryOnDelete, ch.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update channel: %w", err)
	}

	return nil
}

// Delete deletes a channel. If retainHistory is false, also deletes all messages.
func (r *ChannelRepository) Delete(ctx context.Context, id int64, retainHistory bool) error {
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		if !retainHistory {
			// Delete messages first (triggers will handle FTS cleanup)
			if _, err := tx.ExecContext(ctx, "DELETE FROM messages WHERE channel_id = ?", id); err != nil {
				return fmt.Errorf("failed to delete channel messages: %w", err)
			}
		}

		// Delete the channel
		if _, err := tx.ExecContext(ctx, "DELETE FROM channels WHERE id = ?", id); err != nil {
			return fmt.Errorf("failed to delete channel: %w", err)
		}

		return nil
	})
}

// UpdateStats updates the channel message count and last message time.
func (r *ChannelRepository) UpdateStats(ctx context.Context, id int64, totalMessages int64, lastMessageAt time.Time) error {
	query := `
		UPDATE channels
		SET total_messages = ?,
		    last_message_at = ?,
		    updated_at = datetime('now')
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query, totalMessages, lastMessageAt.Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("failed to update channel stats: %w", err)
	}

	return nil
}

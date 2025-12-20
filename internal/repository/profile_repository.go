// Package repository provides database access for the Twitch Chat Archiver.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Profile represents a real-world identity grouping (person/company).
type Profile struct {
	ID          int64
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ProfileChannelLink represents a channel linked to a profile.
type ProfileChannelLink struct {
	ProfileID   int64
	ChannelID   int64
	CreatedAt   time.Time
	Channel     *Channel
	Profile     *Profile
	ChannelName string
}

// ProfileRepository provides CRUD operations for profiles and their channel links.
type ProfileRepository struct {
	db Database
}

// NewProfileRepository creates a new profile repository.
func NewProfileRepository(db Database) *ProfileRepository {
	return &ProfileRepository{db: db}
}

// List returns all profiles.
func (r *ProfileRepository) List(ctx context.Context) ([]Profile, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM profiles
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query profiles: %w", err)
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		var p Profile
		var createdAt, updatedAt any
		var description sql.NullString

		if err := rows.Scan(&p.ID, &p.Name, &description, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan profile: %w", err)
		}
		if description.Valid {
			p.Description = description.String
		}
		p.CreatedAt = parseTimeValue(createdAt)
		p.UpdatedAt = parseTimeValue(updatedAt)
		profiles = append(profiles, p)
	}

	return profiles, rows.Err()
}

// GetByID returns a profile by ID.
func (r *ProfileRepository) GetByID(ctx context.Context, id int64) (*Profile, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM profiles
		WHERE id = ` + r.db.Placeholder(1)

	row := r.db.QueryRowContext(ctx, query, id)
	p, err := scanProfile(row)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Create creates a new profile.
func (r *ProfileRepository) Create(ctx context.Context, p *Profile) error {
	var description sql.NullString
	if p.Description != "" {
		description = sql.NullString{String: p.Description, Valid: true}
	}

	if r.db.SupportsReturning() {
		query := `
			INSERT INTO profiles (name, description)
			VALUES ($1, $2)
			RETURNING id, created_at, updated_at
		`
		var createdAt, updatedAt time.Time
		if err := r.db.QueryRowContext(ctx, query, p.Name, description).Scan(&p.ID, &createdAt, &updatedAt); err != nil {
			return MapSQLError(fmt.Errorf("failed to create profile: %w", err))
		}
		p.CreatedAt = createdAt
		p.UpdatedAt = updatedAt
		return nil
	}

	query := `
		INSERT INTO profiles (name, description, created_at, updated_at)
		VALUES (?, ?, datetime('now'), datetime('now'))
	`
	result, err := r.db.ExecContext(ctx, query, p.Name, description)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to create profile: %w", err))
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	p.ID = id

	created, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if created != nil {
		p.CreatedAt = created.CreatedAt
		p.UpdatedAt = created.UpdatedAt
	}

	return nil
}

// Update updates profile metadata.
func (r *ProfileRepository) Update(ctx context.Context, p *Profile) error {
	var description sql.NullString
	if p.Description != "" {
		description = sql.NullString{String: p.Description, Valid: true}
	}

	var query string
	if r.db.DriverName() == "postgres" {
		query = `
			UPDATE profiles
			SET name = $1,
			    description = $2,
			    updated_at = NOW()
			WHERE id = $3
		`
	} else {
		query = `
			UPDATE profiles
			SET name = ?,
			    description = ?,
			    updated_at = datetime('now')
			WHERE id = ?
		`
	}

	result, err := r.db.ExecContext(ctx, query, p.Name, description, p.ID)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to update profile: %w", err))
	}
	if err := MapResultNotFound(result); err != nil {
		return err
	}

	return nil
}

// Delete deletes a profile.
func (r *ProfileRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM profiles WHERE id = ` + r.db.Placeholder(1)
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to delete profile: %w", err))
	}
	return MapResultNotFound(result)
}

// LinkChannel links a channel to a profile (a channel can belong to at most one profile).
func (r *ProfileRepository) LinkChannel(ctx context.Context, profileID, channelID int64) error {
	querySQLite := `
		INSERT INTO profile_channels (profile_id, channel_id, created_at)
		VALUES (?, ?, datetime('now'))
	`
	queryPostgres := `
		INSERT INTO profile_channels (profile_id, channel_id)
		VALUES ($1, $2)
	`

	query := querySQLite
	if r.db.DriverName() == "postgres" {
		query = queryPostgres
	}

	_, err := r.db.ExecContext(ctx, query, profileID, channelID)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to link channel to profile: %w", err))
	}
	return nil
}

// UnlinkChannel removes a channel link from a profile.
func (r *ProfileRepository) UnlinkChannel(ctx context.Context, profileID, channelID int64) error {
	query := `
		DELETE FROM profile_channels
		WHERE profile_id = ` + r.db.Placeholder(1) + ` AND channel_id = ` + r.db.Placeholder(2)

	result, err := r.db.ExecContext(ctx, query, profileID, channelID)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to unlink channel from profile: %w", err))
	}
	return MapResultNotFound(result)
}

// ListLinkedChannels returns channels linked to a profile.
func (r *ProfileRepository) ListLinkedChannels(ctx context.Context, profileID int64) ([]Channel, error) {
	query := `
		SELECT c.id, c.name, c.display_name, c.enabled, c.retain_history_on_delete,
		       c.created_at, c.updated_at, c.last_message_at, c.total_messages
		FROM profile_channels pc
		JOIN channels c ON pc.channel_id = c.id
		WHERE pc.profile_id = ` + r.db.Placeholder(1) + `
		ORDER BY c.name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to query profile channels: %w", err)
	}
	defer rows.Close()

	// reuse scanChannels logic by duplicating minimal logic
	var channels []Channel
	for rows.Next() {
		var ch Channel
		var createdAt, updatedAt any
		var lastMessageAt any
		if err := rows.Scan(
			&ch.ID, &ch.Name, &ch.DisplayName, &ch.Enabled, &ch.RetainHistoryOnDelete,
			&createdAt, &updatedAt, &lastMessageAt, &ch.TotalMessages,
		); err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}
		ch.CreatedAt = parseTimeValue(createdAt)
		ch.UpdatedAt = parseTimeValue(updatedAt)
		if t := parseTimeValue(lastMessageAt); !t.IsZero() {
			ch.LastMessageAt = &t
		}
		channels = append(channels, ch)
	}

	return channels, rows.Err()
}

// GetProfileByChannelID returns the profile associated with a channel, if any.
func (r *ProfileRepository) GetProfileByChannelID(ctx context.Context, channelID int64) (*Profile, error) {
	query := `
		SELECT p.id, p.name, p.description, p.created_at, p.updated_at
		FROM profile_channels pc
		JOIN profiles p ON pc.profile_id = p.id
		WHERE pc.channel_id = ` + r.db.Placeholder(1)

	row := r.db.QueryRowContext(ctx, query, channelID)
	p, err := scanProfile(row)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func scanProfile(row *sql.Row) (*Profile, error) {
	var p Profile
	var description sql.NullString
	var createdAt, updatedAt any
	if err := row.Scan(&p.ID, &p.Name, &description, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan profile: %w", err)
	}
	if description.Valid {
		p.Description = description.String
	}
	p.CreatedAt = parseTimeValue(createdAt)
	p.UpdatedAt = parseTimeValue(updatedAt)
	return &p, nil
}

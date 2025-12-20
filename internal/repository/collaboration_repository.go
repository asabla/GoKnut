// Package repository provides database access for the Twitch Chat Archiver.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Collaboration represents an ongoing or ad-hoc collaboration between profiles.
type Collaboration struct {
	ID          int64
	Name        string
	Description string
	SharedChat  bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CollaborationRepository provides CRUD operations for collaborations and participants.
type CollaborationRepository struct {
	db Database
}

// NewCollaborationRepository creates a new collaboration repository.
func NewCollaborationRepository(db Database) *CollaborationRepository {
	return &CollaborationRepository{db: db}
}

// List returns all collaborations.
func (r *CollaborationRepository) List(ctx context.Context) ([]Collaboration, error) {
	query := `
		SELECT id, name, description, shared_chat, created_at, updated_at
		FROM collaborations
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query collaborations: %w", err)
	}
	defer rows.Close()

	var collabs []Collaboration
	for rows.Next() {
		c, err := scanCollaborationRows(rows)
		if err != nil {
			return nil, err
		}
		collabs = append(collabs, *c)
	}
	return collabs, rows.Err()
}

// GetByID returns a collaboration by ID.
func (r *CollaborationRepository) GetByID(ctx context.Context, id int64) (*Collaboration, error) {
	query := `
		SELECT id, name, description, shared_chat, created_at, updated_at
		FROM collaborations
		WHERE id = ` + r.db.Placeholder(1)

	row := r.db.QueryRowContext(ctx, query, id)
	c, err := scanCollaborationRow(row)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Create creates a new collaboration.
func (r *CollaborationRepository) Create(ctx context.Context, c *Collaboration) error {
	var description sql.NullString
	if c.Description != "" {
		description = sql.NullString{String: c.Description, Valid: true}
	}

	if r.db.SupportsReturning() {
		query := `
			INSERT INTO collaborations (name, description, shared_chat)
			VALUES ($1, $2, $3)
			RETURNING id, created_at, updated_at
		`
		var createdAt, updatedAt time.Time
		if err := r.db.QueryRowContext(ctx, query, c.Name, description, c.SharedChat).Scan(&c.ID, &createdAt, &updatedAt); err != nil {
			return MapSQLError(fmt.Errorf("failed to create collaboration: %w", err))
		}
		c.CreatedAt = createdAt
		c.UpdatedAt = updatedAt
		return nil
	}

	sharedChatVal := 0
	if c.SharedChat {
		sharedChatVal = 1
	}
	query := `
		INSERT INTO collaborations (name, description, shared_chat, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))
	`
	result, err := r.db.ExecContext(ctx, query, c.Name, description, sharedChatVal)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to create collaboration: %w", err))
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	c.ID = id

	created, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if created != nil {
		c.CreatedAt = created.CreatedAt
		c.UpdatedAt = created.UpdatedAt
	}
	return nil
}

// Update updates collaboration metadata.
func (r *CollaborationRepository) Update(ctx context.Context, c *Collaboration) error {
	var description sql.NullString
	if c.Description != "" {
		description = sql.NullString{String: c.Description, Valid: true}
	}

	var query string
	if r.db.DriverName() == "postgres" {
		query = `
			UPDATE collaborations
			SET name = $1,
			    description = $2,
			    shared_chat = $3,
			    updated_at = NOW()
			WHERE id = $4
		`
		result, err := r.db.ExecContext(ctx, query, c.Name, description, c.SharedChat, c.ID)
		if err != nil {
			return MapSQLError(fmt.Errorf("failed to update collaboration: %w", err))
		}
		if err := MapResultNotFound(result); err != nil {
			return err
		}
		return nil
	}

	sharedChatVal := 0
	if c.SharedChat {
		sharedChatVal = 1
	}

	query = `
		UPDATE collaborations
		SET name = ?,
		    description = ?,
		    shared_chat = ?,
		    updated_at = datetime('now')
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, c.Name, description, sharedChatVal, c.ID)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to update collaboration: %w", err))
	}
	if err := MapResultNotFound(result); err != nil {
		return err
	}
	return nil
}

// Delete deletes a collaboration.
func (r *CollaborationRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM collaborations WHERE id = ` + r.db.Placeholder(1)
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to delete collaboration: %w", err))
	}
	return MapResultNotFound(result)
}

// AddParticipant adds a profile participant to a collaboration.
func (r *CollaborationRepository) AddParticipant(ctx context.Context, collaborationID, profileID int64) error {
	querySQLite := `
		INSERT INTO collaboration_participants (collaboration_id, profile_id, created_at)
		VALUES (?, ?, datetime('now'))
	`
	queryPostgres := `
		INSERT INTO collaboration_participants (collaboration_id, profile_id)
		VALUES ($1, $2)
	`

	query := querySQLite
	if r.db.DriverName() == "postgres" {
		query = queryPostgres
	}

	_, err := r.db.ExecContext(ctx, query, collaborationID, profileID)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to add collaboration participant: %w", err))
	}
	return nil
}

// RemoveParticipant removes a profile from a collaboration.
func (r *CollaborationRepository) RemoveParticipant(ctx context.Context, collaborationID, profileID int64) error {
	query := `
		DELETE FROM collaboration_participants
		WHERE collaboration_id = ` + r.db.Placeholder(1) + ` AND profile_id = ` + r.db.Placeholder(2)

	result, err := r.db.ExecContext(ctx, query, collaborationID, profileID)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to remove collaboration participant: %w", err))
	}
	return MapResultNotFound(result)
}

// ListParticipants returns profiles participating in a collaboration.
func (r *CollaborationRepository) ListParticipants(ctx context.Context, collaborationID int64) ([]Profile, error) {
	query := `
		SELECT p.id, p.name, p.description, p.created_at, p.updated_at
		FROM collaboration_participants cp
		JOIN profiles p ON cp.profile_id = p.id
		WHERE cp.collaboration_id = ` + r.db.Placeholder(1) + `
		ORDER BY p.name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, collaborationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query collaboration participants: %w", err)
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		var p Profile
		var desc sql.NullString
		var createdAt, updatedAt any
		if err := rows.Scan(&p.ID, &p.Name, &desc, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan profile: %w", err)
		}
		if desc.Valid {
			p.Description = desc.String
		}
		p.CreatedAt = parseTimeValue(createdAt)
		p.UpdatedAt = parseTimeValue(updatedAt)
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

// ListCollaborationsForProfile returns collaborations a profile participates in.
func (r *CollaborationRepository) ListCollaborationsForProfile(ctx context.Context, profileID int64) ([]Collaboration, error) {
	query := `
		SELECT c.id, c.name, c.description, c.shared_chat, c.created_at, c.updated_at
		FROM collaboration_participants cp
		JOIN collaborations c ON cp.collaboration_id = c.id
		WHERE cp.profile_id = ` + r.db.Placeholder(1) + `
		ORDER BY c.name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to query profile collaborations: %w", err)
	}
	defer rows.Close()

	var collabs []Collaboration
	for rows.Next() {
		c, err := scanCollaborationRows(rows)
		if err != nil {
			return nil, err
		}
		collabs = append(collabs, *c)
	}
	return collabs, rows.Err()
}

func scanCollaborationRow(row *sql.Row) (*Collaboration, error) {
	var c Collaboration
	var desc sql.NullString
	var createdAt, updatedAt any
	var shared any
	if err := row.Scan(&c.ID, &c.Name, &desc, &shared, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan collaboration: %w", err)
	}
	if desc.Valid {
		c.Description = desc.String
	}
	c.SharedChat = parseBoolValue(shared)
	c.CreatedAt = parseTimeValue(createdAt)
	c.UpdatedAt = parseTimeValue(updatedAt)
	return &c, nil
}

type collaborationRowScanner interface {
	Scan(dest ...any) error
}

func scanCollaborationRows(row collaborationRowScanner) (*Collaboration, error) {
	var c Collaboration
	var desc sql.NullString
	var createdAt, updatedAt any
	var shared any
	if err := row.Scan(&c.ID, &c.Name, &desc, &shared, &createdAt, &updatedAt); err != nil {
		return nil, fmt.Errorf("failed to scan collaboration: %w", err)
	}
	if desc.Valid {
		c.Description = desc.String
	}
	c.SharedChat = parseBoolValue(shared)
	c.CreatedAt = parseTimeValue(createdAt)
	c.UpdatedAt = parseTimeValue(updatedAt)
	return &c, nil
}

func parseBoolValue(v any) bool {
	if v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case int64:
		return t != 0
	case int:
		return t != 0
	case int32:
		return t != 0
	case string:
		return t == "1" || t == "t" || t == "true" || t == "TRUE"
	default:
		return false
	}
}

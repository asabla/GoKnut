// Package repository provides database access for the Twitch Chat Archiver.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Event represents a curated collaboration with time bounds.
type Event struct {
	ID          int64
	Title       string
	Description string
	StartAt     time.Time
	EndAt       *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// EventRepository provides CRUD operations for events and participants.
type EventRepository struct {
	db Database
}

// NewEventRepository creates a new event repository.
func NewEventRepository(db Database) *EventRepository {
	return &EventRepository{db: db}
}

// List returns all events ordered by start time descending.
func (r *EventRepository) List(ctx context.Context) ([]Event, error) {
	query := `
		SELECT id, title, description, start_at, end_at, created_at, updated_at
		FROM events
		ORDER BY start_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		evt, err := scanEventRows(r.db.DriverName(), rows)
		if err != nil {
			return nil, err
		}
		events = append(events, *evt)
	}
	return events, rows.Err()
}

// GetByID returns an event by ID.
func (r *EventRepository) GetByID(ctx context.Context, id int64) (*Event, error) {
	query := `
		SELECT id, title, description, start_at, end_at, created_at, updated_at
		FROM events
		WHERE id = ` + r.db.Placeholder(1)

	row := r.db.QueryRowContext(ctx, query, id)
	evt, err := scanEventRow(r.db.DriverName(), row)
	if err != nil {
		return nil, err
	}
	return evt, nil
}

// Create creates a new event.
func (r *EventRepository) Create(ctx context.Context, evt *Event) error {
	var description sql.NullString
	if evt.Description != "" {
		description = sql.NullString{String: evt.Description, Valid: true}
	}
	var endAt any
	if evt.EndAt != nil && !evt.EndAt.IsZero() {
		if r.db.DriverName() == "postgres" {
			endAt = *evt.EndAt
		} else {
			endAt = evt.EndAt.Format(time.RFC3339)
		}
	} else {
		endAt = sql.NullString{Valid: false}
	}

	if r.db.SupportsReturning() {
		query := `
			INSERT INTO events (title, description, start_at, end_at)
			VALUES ($1, $2, $3, $4)
			RETURNING id, created_at, updated_at
		`
		var createdAt, updatedAt time.Time
		if err := r.db.QueryRowContext(ctx, query, evt.Title, description, evt.StartAt, endAt).Scan(&evt.ID, &createdAt, &updatedAt); err != nil {
			return MapSQLError(fmt.Errorf("failed to create event: %w", err))
		}
		evt.CreatedAt = createdAt
		evt.UpdatedAt = updatedAt
		return nil
	}

	query := `
		INSERT INTO events (title, description, start_at, end_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
	`
	startAtStr := evt.StartAt.Format(time.RFC3339)

	result, err := r.db.ExecContext(ctx, query, evt.Title, description, startAtStr, endAt)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to create event: %w", err))
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	evt.ID = id

	created, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if created != nil {
		evt.CreatedAt = created.CreatedAt
		evt.UpdatedAt = created.UpdatedAt
	}
	return nil
}

// Update updates event metadata and dates.
func (r *EventRepository) Update(ctx context.Context, evt *Event) error {
	var description sql.NullString
	if evt.Description != "" {
		description = sql.NullString{String: evt.Description, Valid: true}
	}

	var endAt any
	if evt.EndAt != nil && !evt.EndAt.IsZero() {
		if r.db.DriverName() == "postgres" {
			endAt = *evt.EndAt
		} else {
			endAt = evt.EndAt.Format(time.RFC3339)
		}
	} else {
		endAt = sql.NullString{Valid: false}
	}

	var query string
	var startAt any
	if r.db.DriverName() == "postgres" {
		query = `
			UPDATE events
			SET title = $1,
			    description = $2,
			    start_at = $3,
			    end_at = $4,
			    updated_at = NOW()
			WHERE id = $5
		`
		startAt = evt.StartAt
	} else {
		query = `
			UPDATE events
			SET title = ?,
			    description = ?,
			    start_at = ?,
			    end_at = ?,
			    updated_at = datetime('now')
			WHERE id = ?
		`
		startAt = evt.StartAt.Format(time.RFC3339)
	}

	result, err := r.db.ExecContext(ctx, query, evt.Title, description, startAt, endAt, evt.ID)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to update event: %w", err))
	}
	if err := MapResultNotFound(result); err != nil {
		return err
	}
	return nil
}

// Delete deletes an event.
func (r *EventRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM events WHERE id = ` + r.db.Placeholder(1)
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to delete event: %w", err))
	}
	return MapResultNotFound(result)
}

// AddParticipant adds a profile participant to an event.
func (r *EventRepository) AddParticipant(ctx context.Context, eventID, profileID int64) error {
	querySQLite := `
		INSERT INTO event_participants (event_id, profile_id, created_at)
		VALUES (?, ?, datetime('now'))
	`
	queryPostgres := `
		INSERT INTO event_participants (event_id, profile_id)
		VALUES ($1, $2)
	`
	query := querySQLite
	if r.db.DriverName() == "postgres" {
		query = queryPostgres
	}

	_, err := r.db.ExecContext(ctx, query, eventID, profileID)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to add event participant: %w", err))
	}
	return nil
}

// RemoveParticipant removes a profile from an event.
func (r *EventRepository) RemoveParticipant(ctx context.Context, eventID, profileID int64) error {
	query := `
		DELETE FROM event_participants
		WHERE event_id = ` + r.db.Placeholder(1) + ` AND profile_id = ` + r.db.Placeholder(2)
	result, err := r.db.ExecContext(ctx, query, eventID, profileID)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to remove event participant: %w", err))
	}
	return MapResultNotFound(result)
}

// ListParticipants returns profiles participating in an event.
func (r *EventRepository) ListParticipants(ctx context.Context, eventID int64) ([]Profile, error) {
	query := `
		SELECT p.id, p.name, p.description, p.created_at, p.updated_at
		FROM event_participants ep
		JOIN profiles p ON ep.profile_id = p.id
		WHERE ep.event_id = ` + r.db.Placeholder(1) + `
		ORDER BY p.name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to query event participants: %w", err)
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		var p Profile
		var description sql.NullString
		var createdAt, updatedAt any
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

// ListEventsForProfile returns events a profile participates in.
func (r *EventRepository) ListEventsForProfile(ctx context.Context, profileID int64) ([]Event, error) {
	query := `
		SELECT e.id, e.title, e.description, e.start_at, e.end_at, e.created_at, e.updated_at
		FROM event_participants ep
		JOIN events e ON ep.event_id = e.id
		WHERE ep.profile_id = ` + r.db.Placeholder(1) + `
		ORDER BY e.start_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to query profile events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		evt, err := scanEventRows(r.db.DriverName(), rows)
		if err != nil {
			return nil, err
		}
		events = append(events, *evt)
	}
	return events, rows.Err()
}

func scanEventRow(driver string, row *sql.Row) (*Event, error) {
	var evt Event
	var description sql.NullString
	var startAt, createdAt, updatedAt any
	var endAt any

	if err := row.Scan(&evt.ID, &evt.Title, &description, &startAt, &endAt, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan event: %w", err)
	}
	if description.Valid {
		evt.Description = description.String
	}

	evt.StartAt = parseTimeValue(startAt)
	if t := parseTimeValue(endAt); !t.IsZero() {
		evt.EndAt = &t
	}
	if driver == "postgres" {
		evt.CreatedAt = parseTimeValue(createdAt)
		evt.UpdatedAt = parseTimeValue(updatedAt)
	} else {
		evt.CreatedAt = parseTimeValue(createdAt)
		evt.UpdatedAt = parseTimeValue(updatedAt)
	}
	return &evt, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEventRows(driver string, row rowScanner) (*Event, error) {
	var evt Event
	var description sql.NullString
	var startAt, createdAt, updatedAt any
	var endAt any

	if err := row.Scan(&evt.ID, &evt.Title, &description, &startAt, &endAt, &createdAt, &updatedAt); err != nil {
		return nil, fmt.Errorf("failed to scan event: %w", err)
	}
	if description.Valid {
		evt.Description = description.String
	}
	evt.StartAt = parseTimeValue(startAt)
	if t := parseTimeValue(endAt); !t.IsZero() {
		evt.EndAt = &t
	}
	evt.CreatedAt = parseTimeValue(createdAt)
	evt.UpdatedAt = parseTimeValue(updatedAt)
	_ = driver
	return &evt, nil
}

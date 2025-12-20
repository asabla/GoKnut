// Package repository provides database access for the Twitch Chat Archiver.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Organization represents a grouping/affiliation.
type Organization struct {
	ID          int64
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// OrganizationRepository provides CRUD operations for organizations and memberships.
type OrganizationRepository struct {
	db Database
}

// NewOrganizationRepository creates a new organization repository.
func NewOrganizationRepository(db Database) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// List returns all organizations.
func (r *OrganizationRepository) List(ctx context.Context) ([]Organization, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM organizations
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query organizations: %w", err)
	}
	defer rows.Close()

	var orgs []Organization
	for rows.Next() {
		var org Organization
		var createdAt, updatedAt any
		var description sql.NullString
		if err := rows.Scan(&org.ID, &org.Name, &description, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan organization: %w", err)
		}
		if description.Valid {
			org.Description = description.String
		}
		org.CreatedAt = parseTimeValue(createdAt)
		org.UpdatedAt = parseTimeValue(updatedAt)
		orgs = append(orgs, org)
	}
	return orgs, rows.Err()
}

// GetByID returns an organization by ID.
func (r *OrganizationRepository) GetByID(ctx context.Context, id int64) (*Organization, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM organizations
		WHERE id = ` + r.db.Placeholder(1)
	row := r.db.QueryRowContext(ctx, query, id)
	return scanOrganization(row)
}

// Create creates a new organization.
func (r *OrganizationRepository) Create(ctx context.Context, org *Organization) error {
	var description sql.NullString
	if org.Description != "" {
		description = sql.NullString{String: org.Description, Valid: true}
	}

	if r.db.SupportsReturning() {
		query := `
			INSERT INTO organizations (name, description)
			VALUES ($1, $2)
			RETURNING id, created_at, updated_at
		`
		var createdAt, updatedAt time.Time
		if err := r.db.QueryRowContext(ctx, query, org.Name, description).Scan(&org.ID, &createdAt, &updatedAt); err != nil {
			return MapSQLError(fmt.Errorf("failed to create organization: %w", err))
		}
		org.CreatedAt = createdAt
		org.UpdatedAt = updatedAt
		return nil
	}

	query := `
		INSERT INTO organizations (name, description, created_at, updated_at)
		VALUES (?, ?, datetime('now'), datetime('now'))
	`
	result, err := r.db.ExecContext(ctx, query, org.Name, description)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to create organization: %w", err))
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	org.ID = id

	created, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if created != nil {
		org.CreatedAt = created.CreatedAt
		org.UpdatedAt = created.UpdatedAt
	}
	return nil
}

// Update updates organization metadata.
func (r *OrganizationRepository) Update(ctx context.Context, org *Organization) error {
	var description sql.NullString
	if org.Description != "" {
		description = sql.NullString{String: org.Description, Valid: true}
	}

	var query string
	if r.db.DriverName() == "postgres" {
		query = `
			UPDATE organizations
			SET name = $1,
			    description = $2,
			    updated_at = NOW()
			WHERE id = $3
		`
	} else {
		query = `
			UPDATE organizations
			SET name = ?,
			    description = ?,
			    updated_at = datetime('now')
			WHERE id = ?
		`
	}

	result, err := r.db.ExecContext(ctx, query, org.Name, description, org.ID)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to update organization: %w", err))
	}
	if err := MapResultNotFound(result); err != nil {
		return err
	}
	return nil
}

// Delete deletes an organization.
func (r *OrganizationRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM organizations WHERE id = ` + r.db.Placeholder(1)
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to delete organization: %w", err))
	}
	return MapResultNotFound(result)
}

// AddMember adds a profile membership to an organization.
func (r *OrganizationRepository) AddMember(ctx context.Context, organizationID, profileID int64) error {
	querySQLite := `
		INSERT INTO organization_members (organization_id, profile_id, created_at)
		VALUES (?, ?, datetime('now'))
	`
	queryPostgres := `
		INSERT INTO organization_members (organization_id, profile_id)
		VALUES ($1, $2)
	`

	query := querySQLite
	if r.db.DriverName() == "postgres" {
		query = queryPostgres
	}

	_, err := r.db.ExecContext(ctx, query, organizationID, profileID)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to add organization member: %w", err))
	}
	return nil
}

// RemoveMember removes a profile membership from an organization.
func (r *OrganizationRepository) RemoveMember(ctx context.Context, organizationID, profileID int64) error {
	query := `
		DELETE FROM organization_members
		WHERE organization_id = ` + r.db.Placeholder(1) + ` AND profile_id = ` + r.db.Placeholder(2)

	result, err := r.db.ExecContext(ctx, query, organizationID, profileID)
	if err != nil {
		return MapSQLError(fmt.Errorf("failed to remove organization member: %w", err))
	}
	return MapResultNotFound(result)
}

// ListMembers returns profiles that belong to an organization.
func (r *OrganizationRepository) ListMembers(ctx context.Context, organizationID int64) ([]Profile, error) {
	query := `
		SELECT p.id, p.name, p.description, p.created_at, p.updated_at
		FROM organization_members om
		JOIN profiles p ON om.profile_id = p.id
		WHERE om.organization_id = ` + r.db.Placeholder(1) + `
		ORDER BY p.name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query organization members: %w", err)
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

// ListOrganizationsForProfile returns organizations a profile belongs to.
func (r *OrganizationRepository) ListOrganizationsForProfile(ctx context.Context, profileID int64) ([]Organization, error) {
	query := `
		SELECT o.id, o.name, o.description, o.created_at, o.updated_at
		FROM organization_members om
		JOIN organizations o ON om.organization_id = o.id
		WHERE om.profile_id = ` + r.db.Placeholder(1) + `
		ORDER BY o.name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to query organization memberships: %w", err)
	}
	defer rows.Close()

	orgs := make([]Organization, 0)
	for rows.Next() {
		var org Organization
		var description sql.NullString
		var createdAt, updatedAt any
		if err := rows.Scan(&org.ID, &org.Name, &description, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan organization: %w", err)
		}
		if description.Valid {
			org.Description = description.String
		}
		org.CreatedAt = parseTimeValue(createdAt)
		org.UpdatedAt = parseTimeValue(updatedAt)
		orgs = append(orgs, org)
	}
	return orgs, rows.Err()
}

func scanOrganization(row *sql.Row) (*Organization, error) {
	var org Organization
	var description sql.NullString
	var createdAt, updatedAt any
	if err := row.Scan(&org.ID, &org.Name, &description, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan organization: %w", err)
	}
	if description.Valid {
		org.Description = description.String
	}
	org.CreatedAt = parseTimeValue(createdAt)
	org.UpdatedAt = parseTimeValue(updatedAt)
	return &org, nil
}

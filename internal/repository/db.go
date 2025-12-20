// Package repository provides database access for the Twitch Chat Archiver.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"modernc.org/sqlite"
)

// Database provides a common interface for database operations.
// Both SQLite and Postgres implementations satisfy this interface.
type Database interface {
	// Query methods
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)

	// Transaction methods
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	WithTx(ctx context.Context, fn func(*sql.Tx) error) error

	// Lifecycle
	Close() error
	Migrate(ctx context.Context) error

	// Driver info
	DriverName() string
	SupportsLastInsertID() bool
	SupportsReturning() bool

	// SQL dialect helpers
	Placeholder(index int) string // Returns $1 for Postgres, ? for SQLite
	NowFunc() string              // Returns NOW() for Postgres, datetime('now') for SQLite
}

// ParseDatetime parses a datetime string from the database.
// Handles both SQLite and Postgres datetime formats.
func ParseDatetime(s string, driver string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	// Common formats to try
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",        // SQLite datetime
		"2006-01-02T15:04:05Z",       // ISO 8601
		"2006-01-02T15:04:05.999999", // Postgres timestamp
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse datetime: %s", s)
}

// OpenDatabase opens a database connection based on the driver and config.
// For SQLite, use driver="sqlite" and dsn=path to db file.
// For Postgres, use driver="postgres" and dsn=connection string.
func OpenDatabase(driver, dsn string, enableFTS bool) (Database, error) {
	switch driver {
	case "sqlite":
		return OpenSQLite(SQLiteDBConfig{
			Path:      dsn,
			EnableFTS: enableFTS,
		})
	case "postgres":
		// Parse DSN components - for now expect full connection string
		cfg := PostgresDBConfig{}
		// Parse the DSN - it should be in the format from config.PostgresDSN()
		_, err := fmt.Sscanf(dsn, "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			&cfg.Host, &cfg.Port, &cfg.User, &cfg.Password, &cfg.Database, &cfg.SSLMode)
		if err != nil {
			// Try opening directly with the DSN
			return OpenPostgresWithDSN(dsn)
		}
		return OpenPostgres(cfg)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}
}

// OpenPostgresWithDSN opens a Postgres connection with a raw DSN string.
func OpenPostgresWithDSN(dsn string) (*PostgresDB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}

	return &PostgresDB{DB: db}, nil
}

var (
	ErrNotFound    = errors.New("not found")
	ErrConflict    = errors.New("conflict")
	ErrConstraint  = errors.New("constraint violation")
	ErrInvalidData = errors.New("invalid data")
)

const (
	sqliteConstraintForeignKey = 787
	sqliteConstraintPrimaryKey = 1555
	sqliteConstraintRowID      = 2579
	sqliteConstraintUnique     = 2067
)

// IsUniqueViolation returns true if err represents a unique/primary-key violation.
func IsUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == "23505"
	}

	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		code := sqliteErr.Code()
		switch code {
		case sqliteConstraintUnique, sqliteConstraintPrimaryKey, sqliteConstraintRowID:
			return true
		default:
			return false
		}
	}

	return false
}

// IsForeignKeyViolation returns true if err represents a foreign-key violation.
func IsForeignKeyViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == "23503"
	}

	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		return sqliteErr.Code() == sqliteConstraintForeignKey
	}

	return false
}

// MapSQLError maps driver-specific SQL errors into shared repository errors.
// The returned error wraps the original error.
func MapSQLError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %w", ErrNotFound, err)
	}
	if IsUniqueViolation(err) {
		return fmt.Errorf("%w: %w", ErrConflict, err)
	}
	if IsForeignKeyViolation(err) {
		return fmt.Errorf("%w: %w", ErrConstraint, err)
	}
	return err
}

// MapResultNotFound returns ErrNotFound if no rows were affected.
func MapResultNotFound(result sql.Result) error {
	if result == nil {
		return nil
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

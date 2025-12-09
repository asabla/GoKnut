// Package repository provides PostgreSQL database access for the Twitch Chat Archiver.
package repository

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

//go:embed migrations/postgres/001_init.sql
var postgresInitSQL string

// PostgresDB wraps the PostgreSQL database connection with configuration and helpers.
type PostgresDB struct {
	*sql.DB
	mu sync.RWMutex
}

// PostgresDBConfig holds PostgreSQL database configuration options.
type PostgresDBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// DSN returns the PostgreSQL connection string.
func (cfg PostgresDBConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode)
}

// OpenPostgres creates a new PostgreSQL database connection.
func OpenPostgres(cfg PostgresDBConfig) (*PostgresDB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
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

// Migrate runs PostgreSQL database migrations.
func (db *PostgresDB) Migrate(ctx context.Context) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.ExecContext(ctx, postgresInitSQL)
	if err != nil {
		return fmt.Errorf("failed to run postgres migrations: %w", err)
	}

	return nil
}

// Close closes the PostgreSQL database connection.
func (db *PostgresDB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.DB.Close()
}

// BeginTx starts a new transaction with the given options.
func (db *PostgresDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return db.DB.BeginTx(ctx, opts)
}

// WithTx executes a function within a transaction.
func (db *PostgresDB) WithTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("failed to rollback: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DriverName returns the database driver name.
func (db *PostgresDB) DriverName() string {
	return "postgres"
}

// SupportsLastInsertID returns false for Postgres (use RETURNING instead).
func (db *PostgresDB) SupportsLastInsertID() bool {
	return false
}

// SupportsReturning returns true for Postgres.
func (db *PostgresDB) SupportsReturning() bool {
	return true
}

// Placeholder returns $N for Postgres.
func (db *PostgresDB) Placeholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

// NowFunc returns NOW() for Postgres.
func (db *PostgresDB) NowFunc() string {
	return "NOW()"
}

// ParsePostgresDatetime parses a datetime from Postgres.
func ParsePostgresDatetime(t sql.NullTime) (time.Time, bool) {
	if !t.Valid {
		return time.Time{}, false
	}
	return t.Time, true
}

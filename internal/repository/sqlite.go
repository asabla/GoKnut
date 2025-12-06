// Package repository provides SQLite database access for the Twitch Chat Archiver.
package repository

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

//go:embed migrations/001_init.sql
var initSQL string

// DB wraps the SQLite database connection with configuration and helpers.
type DB struct {
	*sql.DB
	mu sync.RWMutex
}

// DBConfig holds database configuration options.
type DBConfig struct {
	Path      string
	EnableFTS bool
}

// Open creates a new database connection with WAL mode and performance pragmas.
func Open(cfg DBConfig) (*DB, error) {
	// Build connection string with pragmas
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000&_cache_size=-64000", cfg.Path)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite doesn't handle concurrent writes well
	db.SetMaxIdleConns(1)

	// Apply additional pragmas
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA temp_store = MEMORY",
		"PRAGMA mmap_size = 268435456", // 256MB memory-mapped I/O
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set pragma %q: %w", pragma, err)
		}
	}

	return &DB{DB: db}, nil
}

// Migrate runs database migrations.
func (db *DB) Migrate(ctx context.Context) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.ExecContext(ctx, initSQL)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Checkpoint WAL before closing
	_, _ = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")

	return db.DB.Close()
}

// BeginTx starts a new transaction with the given options.
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return db.DB.BeginTx(ctx, opts)
}

// WithTx executes a function within a transaction.
func (db *DB) WithTx(ctx context.Context, fn func(*sql.Tx) error) error {
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

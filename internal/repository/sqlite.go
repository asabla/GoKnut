// Package repository provides SQLite database access for the Twitch Chat Archiver.
package repository

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// SQLite datetime formats
const (
	// SQLiteDatetime is the format used by SQLite's datetime() function
	SQLiteDatetime = "2006-01-02 15:04:05"
)

// ParseSQLiteDatetime parses a datetime string from SQLite.
// It tries multiple formats since SQLite can store dates in various formats.
func ParseSQLiteDatetime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	// Try SQLite datetime format first (most common)
	if t, err := time.Parse(SQLiteDatetime, s); err == nil {
		return t, nil
	}

	// Try RFC3339 as fallback
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try RFC3339Nano as another fallback
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unable to parse datetime: %s", s)
}

//go:embed migrations/001_init.sql
var sqliteInitSQL string

// SQLiteDB wraps the SQLite database connection with configuration and helpers.
type SQLiteDB struct {
	*sql.DB
	mu sync.RWMutex
}

// SQLiteDBConfig holds SQLite database configuration options.
type SQLiteDBConfig struct {
	Path      string
	EnableFTS bool
}

// OpenSQLite creates a new SQLite database connection with WAL mode and performance pragmas.
func OpenSQLite(cfg SQLiteDBConfig) (*SQLiteDB, error) {
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

	return &SQLiteDB{DB: db}, nil
}

// Migrate runs SQLite database migrations.
func (db *SQLiteDB) Migrate(ctx context.Context) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.ExecContext(ctx, sqliteInitSQL)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Close closes the SQLite database connection.
func (db *SQLiteDB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Checkpoint WAL before closing
	_, _ = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")

	return db.DB.Close()
}

// BeginTx starts a new transaction with the given options.
func (db *SQLiteDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return db.DB.BeginTx(ctx, opts)
}

// WithTx executes a function within a transaction.
func (db *SQLiteDB) WithTx(ctx context.Context, fn func(*sql.Tx) error) error {
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
func (db *SQLiteDB) DriverName() string {
	return "sqlite"
}

// SupportsLastInsertID returns true for SQLite.
func (db *SQLiteDB) SupportsLastInsertID() bool {
	return true
}

// SupportsReturning returns false for SQLite (older versions).
func (db *SQLiteDB) SupportsReturning() bool {
	return false
}

// Placeholder returns ? for SQLite.
func (db *SQLiteDB) Placeholder(index int) string {
	return "?"
}

// NowFunc returns datetime('now') for SQLite.
func (db *SQLiteDB) NowFunc() string {
	return "datetime('now')"
}

// Open is a compatibility wrapper that creates a SQLite database.
// Deprecated: Use OpenSQLite directly.
func Open(cfg DBConfig) (*SQLiteDB, error) {
	return OpenSQLite(SQLiteDBConfig{
		Path:      cfg.Path,
		EnableFTS: cfg.EnableFTS,
	})
}

// DBConfig is kept for backward compatibility.
// Deprecated: Use SQLiteDBConfig.
type DBConfig struct {
	Path      string
	EnableFTS bool
}

// DB is an alias for SQLiteDB for backward compatibility.
type DB = SQLiteDB

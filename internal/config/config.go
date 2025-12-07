// Package config provides configuration loading and validation for the Twitch Chat Archiver.
package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

// AuthMode represents the Twitch IRC authentication mode.
type AuthMode string

const (
	// AuthModeAuthenticated uses OAuth token authentication (full access).
	AuthModeAuthenticated AuthMode = "authenticated"
	// AuthModeAnonymous uses justinfan anonymous login (read-only).
	AuthModeAnonymous AuthMode = "anonymous"
)

// Config holds the application configuration.
type Config struct {
	// Database
	DBPath string

	// HTTP Server
	HTTPAddr string

	// Twitch IRC
	TwitchAuthMode   AuthMode // "authenticated" or "anonymous"
	TwitchUsername   string   // Required for authenticated mode
	TwitchOAuthToken string   // Required for authenticated mode (format: oauth:xxx)
	TwitchChannels   []string

	// Ingestion
	BatchSize    int
	FlushTimeout int // milliseconds
	BufferSize   int // ingestion buffer size

	// Feature flags
	EnableFTS bool
	EnableSSE bool // Enable Server-Sent Events for live updates
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		DBPath:         "./twitch.db",
		HTTPAddr:       ":8080",
		TwitchAuthMode: AuthModeAuthenticated, // Default to authenticated
		BatchSize:      100,
		FlushTimeout:   100,
		BufferSize:     10000,
		EnableFTS:      true,
		EnableSSE:      true, // SSE enabled by default
	}
}

// Load reads configuration from flags and environment variables.
// Environment variables override flags. Returns an error if required fields are missing.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Define flags
	flag.StringVar(&cfg.DBPath, "db-path", cfg.DBPath, "Path to SQLite database file")
	flag.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP server listen address")
	flag.IntVar(&cfg.BatchSize, "batch-size", cfg.BatchSize, "Message batch size for ingestion")
	flag.IntVar(&cfg.FlushTimeout, "flush-timeout", cfg.FlushTimeout, "Batch flush timeout in milliseconds")
	flag.IntVar(&cfg.BufferSize, "buffer-size", cfg.BufferSize, "Ingestion buffer size")
	flag.BoolVar(&cfg.EnableFTS, "enable-fts", cfg.EnableFTS, "Enable FTS5 full-text search")

	flag.Parse()

	// Override with environment variables
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		cfg.HTTPAddr = v
	}
	if v := os.Getenv("TWITCH_USERNAME"); v != "" {
		cfg.TwitchUsername = v
	}
	if v := os.Getenv("TWITCH_OAUTH_TOKEN"); v != "" {
		cfg.TwitchOAuthToken = v
	}
	if v := os.Getenv("TWITCH_CHANNELS"); v != "" {
		channels := strings.Split(v, ",")
		for i, ch := range channels {
			channels[i] = strings.TrimSpace(strings.ToLower(ch))
		}
		cfg.TwitchChannels = channels
	}
	if v := os.Getenv("BATCH_SIZE"); v != "" {
		var size int
		if _, err := fmt.Sscanf(v, "%d", &size); err == nil && size > 0 {
			cfg.BatchSize = size
		}
	}
	if v := os.Getenv("FLUSH_TIMEOUT"); v != "" {
		var timeout int
		if _, err := fmt.Sscanf(v, "%d", &timeout); err == nil && timeout > 0 {
			cfg.FlushTimeout = timeout
		}
	}
	if v := os.Getenv("BUFFER_SIZE"); v != "" {
		var size int
		if _, err := fmt.Sscanf(v, "%d", &size); err == nil && size > 0 {
			cfg.BufferSize = size
		}
	}
	if v := os.Getenv("ENABLE_FTS"); v != "" {
		cfg.EnableFTS = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("ENABLE_SSE"); v != "" {
		cfg.EnableSSE = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("TWITCH_AUTH_MODE"); v != "" {
		switch strings.ToLower(v) {
		case "authenticated", "auth":
			cfg.TwitchAuthMode = AuthModeAuthenticated
		case "anonymous", "anon":
			cfg.TwitchAuthMode = AuthModeAnonymous
		default:
			return nil, fmt.Errorf("invalid TWITCH_AUTH_MODE: %s (must be 'authenticated' or 'anonymous')", v)
		}
	}

	// Auto-detect auth mode if not explicitly set: anonymous if no credentials provided
	if os.Getenv("TWITCH_AUTH_MODE") == "" {
		if cfg.TwitchUsername == "" && cfg.TwitchOAuthToken == "" {
			cfg.TwitchAuthMode = AuthModeAnonymous
		}
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that required configuration fields are set.
func (c *Config) Validate() error {
	var errs []string

	if c.DBPath == "" {
		errs = append(errs, "db-path is required")
	}
	if c.HTTPAddr == "" {
		errs = append(errs, "http-addr is required")
	}

	// Auth mode-specific validation
	switch c.TwitchAuthMode {
	case AuthModeAuthenticated:
		if c.TwitchUsername == "" {
			errs = append(errs, "TWITCH_USERNAME is required for authenticated mode")
		}
		if c.TwitchOAuthToken == "" {
			errs = append(errs, "TWITCH_OAUTH_TOKEN is required for authenticated mode")
		} else if !strings.HasPrefix(c.TwitchOAuthToken, "oauth:") {
			errs = append(errs, "TWITCH_OAUTH_TOKEN must start with 'oauth:' prefix")
		}
	case AuthModeAnonymous:
		if c.TwitchOAuthToken != "" {
			errs = append(errs, "TWITCH_OAUTH_TOKEN must not be set for anonymous mode")
		}
		// Username is optional for anonymous mode (will generate justinfan nick)
	default:
		errs = append(errs, fmt.Sprintf("invalid auth mode: %s", c.TwitchAuthMode))
	}

	if c.BatchSize <= 0 {
		errs = append(errs, "batch-size must be positive")
	}
	if c.FlushTimeout <= 0 {
		errs = append(errs, "flush-timeout must be positive")
	}
	if c.BufferSize <= 0 {
		errs = append(errs, "buffer-size must be positive")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

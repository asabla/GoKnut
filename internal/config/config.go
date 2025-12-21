// Package config provides configuration loading and validation for the Twitch Chat Archiver.
package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
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

// DBDriver represents the database driver type.
type DBDriver string

const (
	// DBDriverSQLite uses SQLite as the database backend.
	DBDriverSQLite DBDriver = "sqlite"
	// DBDriverPostgres uses PostgreSQL as the database backend.
	DBDriverPostgres DBDriver = "postgres"
)

// Config holds the application configuration.
type Config struct {
	// Database
	DBDriver DBDriver // "sqlite" or "postgres"
	DBPath   string   // SQLite: path to database file

	// PostgreSQL settings (used when DBDriver == "postgres")
	PGHost     string
	PGPort     int
	PGUser     string
	PGPassword string
	PGDatabase string
	PGSSLMode  string

	// HTTP Server
	HTTPAddr string

	// Prometheus (optional, used for dashboard diagrams)
	PrometheusBaseURL string
	PrometheusTimeout int // milliseconds

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
	EnableFTS bool // FTS5 full-text search (SQLite only)
	EnableSSE bool // Enable Server-Sent Events for live updates

	// OpenTelemetry configuration
	OTelEnabled        bool   // Enable OpenTelemetry instrumentation
	OTelServiceName    string // Service name for traces/metrics
	OTelExporterOTLP   string // OTLP endpoint (e.g., "localhost:4317")
	OTelInsecure       bool   // Use insecure connection to OTLP endpoint
	OTelSamplerRatio   float64
	OTelMetricsEnabled bool
	OTelTracesEnabled  bool
	OTelLogsEnabled    bool
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		// Database defaults
		DBDriver:   DBDriverSQLite,
		DBPath:     "./twitch.db",
		PGHost:     "localhost",
		PGPort:     5432,
		PGUser:     "goknut",
		PGPassword: "goknut",
		PGDatabase: "goknut",
		PGSSLMode:  "disable",

		// HTTP defaults
		HTTPAddr: ":8080",

		// Prometheus defaults
		PrometheusBaseURL: "http://localhost:9090",
		PrometheusTimeout: 1500,

		// Twitch defaults
		TwitchAuthMode: AuthModeAuthenticated,

		// Ingestion defaults
		BatchSize:    100,
		FlushTimeout: 100,
		BufferSize:   10000,

		// Feature flags
		EnableFTS: true,
		EnableSSE: true,

		// OpenTelemetry defaults
		OTelEnabled:        false,
		OTelServiceName:    "goknut",
		OTelExporterOTLP:   "localhost:4317",
		OTelInsecure:       true,
		OTelSamplerRatio:   1.0,
		OTelMetricsEnabled: true,
		OTelTracesEnabled:  true,
		OTelLogsEnabled:    true,
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
	flag.StringVar(&cfg.PrometheusBaseURL, "prometheus-base-url", cfg.PrometheusBaseURL, "Prometheus base URL (optional; used for dashboard diagrams)")
	flag.IntVar(&cfg.PrometheusTimeout, "prometheus-timeout-ms", cfg.PrometheusTimeout, "Prometheus HTTP timeout in milliseconds")

	flag.Parse()

	// Override with environment variables
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		cfg.HTTPAddr = v
	}
	if v := os.Getenv("PROMETHEUS_BASE_URL"); v != "" {
		cfg.PrometheusBaseURL = v
	}
	if v := os.Getenv("PROMETHEUS_TIMEOUT_MS"); v != "" {
		if timeout, err := strconv.Atoi(v); err == nil && timeout > 0 {
			cfg.PrometheusTimeout = timeout
		}
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

	// Database driver selection
	if v := os.Getenv("DB_DRIVER"); v != "" {
		switch strings.ToLower(v) {
		case "sqlite":
			cfg.DBDriver = DBDriverSQLite
		case "postgres", "postgresql":
			cfg.DBDriver = DBDriverPostgres
		default:
			return nil, fmt.Errorf("invalid DB_DRIVER: %s (must be 'sqlite' or 'postgres')", v)
		}
	}

	// PostgreSQL settings
	if v := os.Getenv("PG_HOST"); v != "" {
		cfg.PGHost = v
	}
	if v := os.Getenv("PG_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil && port > 0 {
			cfg.PGPort = port
		}
	}
	if v := os.Getenv("PG_USER"); v != "" {
		cfg.PGUser = v
	}
	if v := os.Getenv("PG_PASSWORD"); v != "" {
		cfg.PGPassword = v
	}
	if v := os.Getenv("PG_DATABASE"); v != "" {
		cfg.PGDatabase = v
	}
	if v := os.Getenv("PG_SSLMODE"); v != "" {
		cfg.PGSSLMode = v
	}

	// OpenTelemetry settings
	if v := os.Getenv("OTEL_ENABLED"); v != "" {
		cfg.OTelEnabled = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("OTEL_SERVICE_NAME"); v != "" {
		cfg.OTelServiceName = v
	}
	if v := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); v != "" {
		cfg.OTelExporterOTLP = v
	}
	if v := os.Getenv("OTEL_INSECURE"); v != "" {
		cfg.OTelInsecure = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("OTEL_SAMPLER_RATIO"); v != "" {
		if ratio, err := strconv.ParseFloat(v, 64); err == nil && ratio >= 0 && ratio <= 1 {
			cfg.OTelSamplerRatio = ratio
		}
	}
	if v := os.Getenv("OTEL_METRICS_ENABLED"); v != "" {
		cfg.OTelMetricsEnabled = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("OTEL_TRACES_ENABLED"); v != "" {
		cfg.OTelTracesEnabled = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("OTEL_LOGS_ENABLED"); v != "" {
		cfg.OTelLogsEnabled = strings.ToLower(v) == "true" || v == "1"
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

	// Database validation based on driver
	switch c.DBDriver {
	case DBDriverSQLite:
		if c.DBPath == "" {
			errs = append(errs, "db-path is required for SQLite")
		}
	case DBDriverPostgres:
		if c.PGHost == "" {
			errs = append(errs, "PG_HOST is required for Postgres")
		}
		if c.PGPort <= 0 {
			errs = append(errs, "PG_PORT must be positive")
		}
		if c.PGUser == "" {
			errs = append(errs, "PG_USER is required for Postgres")
		}
		if c.PGDatabase == "" {
			errs = append(errs, "PG_DATABASE is required for Postgres")
		}
		// Disable FTS for Postgres (SQLite-only feature)
		if c.EnableFTS {
			c.EnableFTS = false
		}
	default:
		errs = append(errs, fmt.Sprintf("invalid DB driver: %s", c.DBDriver))
	}

	if c.HTTPAddr == "" {
		errs = append(errs, "http-addr is required")
	}
	if c.PrometheusTimeout <= 0 {
		errs = append(errs, "prometheus-timeout-ms must be positive")
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

	// OTel validation
	if c.OTelEnabled {
		if c.OTelServiceName == "" {
			errs = append(errs, "OTEL_SERVICE_NAME is required when OTel is enabled")
		}
		if c.OTelExporterOTLP == "" {
			errs = append(errs, "OTEL_EXPORTER_OTLP_ENDPOINT is required when OTel is enabled")
		}
		if c.OTelSamplerRatio < 0 || c.OTelSamplerRatio > 1 {
			errs = append(errs, "OTEL_SAMPLER_RATIO must be between 0 and 1")
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

// PostgresDSN returns the PostgreSQL connection string.
func (c *Config) PostgresDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.PGHost, c.PGPort, c.PGUser, c.PGPassword, c.PGDatabase, c.PGSSLMode)
}

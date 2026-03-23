package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	CFAPIKey      string
	GatewayAPIKey string
	Port          string
	LogLevel      string
}

// SlogLevel converts the LogLevel string to a slog.Level.
func (c *Config) SlogLevel() slog.Level {
	switch strings.ToLower(c.LogLevel) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Load reads configuration from environment variables (and an optional .env file).
// Returns an error if any required variable is missing or any value is invalid.
func Load() (*Config, error) {
	// Load .env if present — ignore error if file doesn't exist
	_ = godotenv.Load()

	cfg := &Config{
		CFAPIKey:      os.Getenv("CF_API_KEY"),
		GatewayAPIKey: os.Getenv("GATEWAY_API_KEY"),
		Port:          os.Getenv("PORT"),
		LogLevel:      os.Getenv("LOG_LEVEL"),
	}

	// CF_API_KEY — required
	if cfg.CFAPIKey == "" {
		return nil, fmt.Errorf("CF_API_KEY is required")
	}

	// GATEWAY_API_KEY — required
	if cfg.GatewayAPIKey == "" {
		return nil, fmt.Errorf("GATEWAY_API_KEY is required")
	}

	// PORT — default 8080
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	// LOG_LEVEL — default info, validate
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	switch strings.ToLower(cfg.LogLevel) {
	case "debug", "info", "warn", "warning", "error":
		// valid
	default:
		return nil, fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error, got %q", cfg.LogLevel)
	}

	return cfg, nil
}

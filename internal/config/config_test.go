package config_test

import (
	"log/slog"
	"testing"

	"github.com/lobo235/curseforge-gateway/internal/config"
)

// setRequired sets all required env vars; individual tests may blank one out.
func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("CF_API_KEY", "test-cf-key")
	t.Setenv("GATEWAY_API_KEY", "key123")
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "")
}

func TestLoad_Defaults(t *testing.T) {
	setRequired(t)
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.CFAPIKey != "test-cf-key" {
		t.Errorf("CFAPIKey = %q, want test-cf-key", cfg.CFAPIKey)
	}
}

func TestLoad_PortOverride(t *testing.T) {
	setRequired(t)
	t.Setenv("PORT", "9090")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want 9090", cfg.Port)
	}
}

func TestLoad_MissingCFAPIKey(t *testing.T) {
	setRequired(t)
	t.Setenv("CF_API_KEY", "")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected error for missing CF_API_KEY")
	}
}

func TestLoad_MissingGatewayAPIKey(t *testing.T) {
	setRequired(t)
	t.Setenv("GATEWAY_API_KEY", "")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected error for missing GATEWAY_API_KEY")
	}
}

func TestLoad_LogLevels(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "warning", "error"} {
		t.Run(level, func(t *testing.T) {
			setRequired(t)
			t.Setenv("LOG_LEVEL", level)
			if _, err := config.Load(); err != nil {
				t.Errorf("LOG_LEVEL=%q should be valid, got: %v", level, err)
			}
		})
	}
}

func TestLoad_InvalidLogLevel(t *testing.T) {
	setRequired(t)
	t.Setenv("LOG_LEVEL", "verbose")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected error for invalid LOG_LEVEL")
	}
}

func TestSlogLevel(t *testing.T) {
	cases := []struct {
		level string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"INFO", slog.LevelInfo},
	}
	for _, tc := range cases {
		setRequired(t)
		t.Setenv("LOG_LEVEL", tc.level)
		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("LOG_LEVEL=%q: unexpected error: %v", tc.level, err)
		}
		if got := cfg.SlogLevel(); got != tc.want {
			t.Errorf("LOG_LEVEL=%q: SlogLevel() = %v, want %v", tc.level, got, tc.want)
		}
	}
}

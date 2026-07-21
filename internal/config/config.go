package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration, loaded from environment variables.
type Config struct {
	TelegramToken string
	DatabaseURL   string
	LogLevel      string
	Env           string // "development" or "production"
}

// Load reads configuration from the environment and validates required fields.
func Load() (Config, error) {
	cfg := Config{
		TelegramToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		DatabaseURL:   DatabaseURL(),
		LogLevel:      getenv("LOG_LEVEL", "info"),
		Env:           getenv("APP_ENV", "development"),
	}

	var errs []error
	if cfg.TelegramToken == "" {
		errs = append(errs, errors.New("TELEGRAM_BOT_TOKEN is required"))
	}
	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL (or POSTGRES_* vars) is required"))
	}
	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// DatabaseURL returns DATABASE_URL when set, otherwise assembles one from the
// individual POSTGRES_* variables that docker-compose provides. It is exported
// so the migrate tool can obtain a connection string without requiring the
// full (token-validated) config.
func DatabaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		return ""
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		getenv("POSTGRES_USER", "postgres"),
		os.Getenv("POSTGRES_PASSWORD"),
		host,
		getenv("POSTGRES_PORT", "5432"),
		getenv("POSTGRES_DB", "postgres"),
		getenv("POSTGRES_SSLMODE", "disable"),
	)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// GetInt reads an integer env var with a fallback (exported helper for callers).
func GetInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

package env

import (
	"log/slog"

	"github.com/joho/godotenv"
)

// Load attempts to load environment variables from .env files.
// It checks the current directory and the project root (via ../../.env).
func Load() {
	// Try local .env
	if err := godotenv.Load(".env"); err != nil {
		// Silent fail if missing, might be provided by host or root .env
	}

	// Try root .env (relative to services/name/)
	if err := godotenv.Load("../../.env"); err != nil {
		// Silent fail if missing
	}

	slog.Debug("env_load_attempt_complete")
}

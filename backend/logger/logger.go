// ABOUTME: Structured logging configuration using log/slog.
// ABOUTME: Provides Init() to configure default logger with level and format from environment.

package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Init configures the default slog logger based on environment variables.
// LOG_LEVEL: debug, info, warn, error (default: info)
// LOG_FORMAT: text, json (default: text)
func Init() {
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	format := strings.ToLower(os.Getenv("LOG_FORMAT"))

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}

// parseLevel converts a string log level to slog.Level.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
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

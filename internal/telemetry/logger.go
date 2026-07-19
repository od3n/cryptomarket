package telemetry

import (
	"log/slog"
	"os"
	"strings"
)

// NewLogger creates a structured JSON logger with the given level.
func NewLogger(level string, serviceName string) *slog.Logger {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})

	logger := slog.New(handler).With(
		slog.String("service", serviceName),
	)

	return logger
}

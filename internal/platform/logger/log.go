package logger

import (
	"log/slog"
	"os"
)

func NewLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo, // Set minimum level
		// Rename standard keys to match Google Cloud Logging schema
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Rename "level" to "severity" for Google Cloud
			if a.Key == slog.LevelKey {
				a.Key = "severity"
			}
			// Rename "msg" to "message" for consistency (optional but recommended)
			if a.Key == slog.MessageKey {
				a.Key = "message"
			}
			return a
		},
	}

	// Use JSONHandler because Cloud Run automatically parses JSON to structured logs
	handler := slog.NewJSONHandler(os.Stdout, opts)

	return slog.New(handler)
}

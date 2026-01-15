package logger

import (
	"log/slog"
	"os"
)

var Log *slog.Logger

func init() {
	// Configure structured logger
	logLevel := os.Getenv("LOG_LEVEL")
	level := slog.LevelInfo

	switch logLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	Log = slog.New(handler)
}

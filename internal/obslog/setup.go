package obslog

import (
	"log/slog"
	"os"
	"strings"
)

// SetupDefault configures JSON logs to stdout for the process (design.md observability).
// LOG_LEVEL: debug, info, warn, error (default info).
func SetupDefault(service string) {
	level := slog.LevelInfo
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(h).With("service", service))
}
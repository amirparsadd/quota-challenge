package main

import (
	"log/slog"
	"os"
	"time"
)

func envDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		slog.Warn("invalid duration env, using default", "key", key, "value", v, "default", def)
		return def
	}
	return d
}
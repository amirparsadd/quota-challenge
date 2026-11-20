package main

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	"quota-challenge/internal/region/postgres"
)

// DB pool tuning (optional; pgx defaults when unset).
const (
	envPoolMaxConns         = "DB_POOL_MAX_CONNS"
	envPoolMinConns         = "DB_POOL_MIN_CONNS"
	envPoolMaxConnLifetime  = "DB_POOL_MAX_CONN_LIFETIME"
	envPoolMaxConnIdleTime  = "DB_POOL_MAX_CONN_IDLE_TIME"
)

func poolSettingsFromEnv() postgres.PoolSettings {
	var s postgres.PoolSettings
	if v := os.Getenv(envPoolMaxConns); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil || n < 1 {
			slog.Warn("invalid pool env, ignored", "key", envPoolMaxConns, "value", v)
		} else {
			s.SetMaxConns = true
			s.MaxConns = int32(n)
		}
	}
	if v := os.Getenv(envPoolMinConns); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil || n < 0 {
			slog.Warn("invalid pool env, ignored", "key", envPoolMinConns, "value", v)
		} else {
			s.SetMinConns = true
			s.MinConns = int32(n)
		}
	}
	if v := os.Getenv(envPoolMaxConnLifetime); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			slog.Warn("invalid pool env, ignored", "key", envPoolMaxConnLifetime, "value", v)
		} else {
			s.SetMaxConnLifetime = true
			s.MaxConnLifetime = d
		}
	}
	if v := os.Getenv(envPoolMaxConnIdleTime); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			slog.Warn("invalid pool env, ignored", "key", envPoolMaxConnIdleTime, "value", v)
		} else {
			s.SetMaxConnIdleTime = true
			s.MaxConnIdleTime = d
		}
	}
	return s
}
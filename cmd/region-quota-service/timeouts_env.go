package main

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	"quota-challenge/internal/region/regionsvc"
)

const (
	envDBQueryTimeout              = "DB_QUERY_TIMEOUT"
	envIdempotencyLeaseTimeout     = "IDEMPOTENCY_LEASE_TIMEOUT"
	envDefaultReservationTTLSeconds = "DEFAULT_RESERVATION_TTL_SECONDS"
)

func regionConfigFromEnv() regionsvc.Config {
	cfg := regionsvc.DefaultConfig()
	cfg.IdempotencyLease = envDuration(envIdempotencyLeaseTimeout, cfg.IdempotencyLease)
	if v := os.Getenv(envDefaultReservationTTLSeconds); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 1 {
			slog.Warn("invalid env, ignored", "key", envDefaultReservationTTLSeconds, "value", v)
		} else {
			cfg.DefaultReservationTTL = n
		}
	}
	return cfg
}

func dbQueryTimeoutFromEnv() time.Duration {
	return envDuration(envDBQueryTimeout, 0)
}
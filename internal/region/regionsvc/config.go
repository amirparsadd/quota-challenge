package regionsvc

import "time"

// Config holds region-service tuning read from environment in cmd/*.
type Config struct {
	IdempotencyLease      time.Duration
	DefaultReservationTTL int64 // seconds; 0 = product catalog default when client omits ttl
}

func DefaultConfig() Config {
	return Config{IdempotencyLease: 15 * time.Second}
}
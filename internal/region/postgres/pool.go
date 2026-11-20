package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolSettings optional pgxpool tuning. Zero value leaves pgx defaults.
type PoolSettings struct {
	MaxConns         int32
	SetMaxConns      bool
	MinConns         int32
	SetMinConns      bool
	MaxConnLifetime  time.Duration
	SetMaxConnLifetime  bool
	MaxConnIdleTime  time.Duration
	SetMaxConnIdleTime  bool
}

func (s PoolSettings) apply(cfg *pgxpool.Config) {
	if s.SetMaxConns {
		cfg.MaxConns = s.MaxConns
	}
	if s.SetMinConns {
		cfg.MinConns = s.MinConns
	}
	if s.SetMaxConnLifetime {
		cfg.MaxConnLifetime = s.MaxConnLifetime
	}
	if s.SetMaxConnIdleTime {
		cfg.MaxConnIdleTime = s.MaxConnIdleTime
	}
}
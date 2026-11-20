package postgres

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"quota-challenge/internal/obslog"
)

func attachQueryTracer(connCfg *pgx.ConnConfig) {
	obslog.AttachPoolTracer(connCfg)
}

// ConfigurePoolTracing attaches SQL debug tracing (used by tests and OpenPool).
func ConfigurePoolTracing(cfg *pgxpool.Config) {
	if cfg != nil && cfg.ConnConfig != nil {
		attachQueryTracer(cfg.ConnConfig)
	}
}
package postgres

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPoolSettingsApply(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://localhost:5432/db?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	beforeMax := cfg.MaxConns

	s := PoolSettings{
		SetMaxConns:           true,
		MaxConns:              12,
		SetMinConns:           true,
		MinConns:              2,
		SetMaxConnLifetime:    true,
		MaxConnLifetime:       time.Hour,
		SetMaxConnIdleTime:    true,
		MaxConnIdleTime:       10 * time.Minute,
	}
	s.apply(cfg)

	if cfg.MaxConns != 12 || cfg.MinConns != 2 {
		t.Fatalf("conns: max=%d min=%d", cfg.MaxConns, cfg.MinConns)
	}
	if cfg.MaxConnLifetime != time.Hour || cfg.MaxConnIdleTime != 10*time.Minute {
		t.Fatalf("lifetime/idle mismatch")
	}

	empty := PoolSettings{}
	empty.apply(cfg)
	if cfg.MaxConns != 12 {
		t.Fatalf("empty apply should not change max conns, got %d", cfg.MaxConns)
	}
	_ = beforeMax
}
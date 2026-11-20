package main

import (
	"context"
	"fmt"
	"log/slog"

	"quota-challenge/internal/obslog"
	"quota-challenge/internal/region/postgres"
	"quota-challenge/internal/region/regionsvc"
)

func runMigrate(args []string) int {
	obslog.SetupDefault("region-quota-service")
	dsn := flagString(args, "--database-url", env("DATABASE_URL", ""))
	if dsn == "" {
		slog.Error("DATABASE_URL required")
		return 1
	}
	if err := postgres.Migrate(context.Background(), dsn); err != nil {
		slog.Error("migrate failed", "err", err)
		return 1
	}
	return 0
}

func runReap(args []string) int {
	obslog.SetupDefault("region-quota-service")
	dsn := env("DATABASE_URL", "")
	pool := mustPool(dsn)
	defer pool.Close()
	svc := regionsvc.New(postgres.NewStore(pool, env("REGION_ID", "ir-thr-1")))
	n, err := svc.Reap(context.Background())
	if err != nil {
		slog.Error("reap", "err", err)
		return 1
	}
	fmt.Println(n)
	return 0
}

func runDrift(args []string) int {
	obslog.SetupDefault("region-quota-service")
	apply := flagBool(args, "--apply")
	dsn := env("DATABASE_URL", "")
	pool := mustPool(dsn)
	defer pool.Close()
	svc := regionsvc.New(postgres.NewStore(pool, env("REGION_ID", "ir-thr-1")))
	drift, err := svc.Resync(context.Background(), apply)
	if err != nil {
		slog.Error("drift", "err", err)
		return 1
	}
	fmt.Println(drift)
	if !apply && drift > 0 {
		return 1
	}
	return 0
}

func runHealthcheck() int {
	dsn := env("DATABASE_URL", "")
	if dsn == "" {
		return 1
	}
	pool, err := postgres.OpenPool(context.Background(), dsn, postgres.PoolSettings{})
	if err != nil {
		return 1
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		return 1
	}
	return 0
}
package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrate(ctx context.Context, databaseURL string) error {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("pool: %w", err)
	}
	defer pool.Close()

	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	goose.SetBaseFS(migrations)
	return goose.UpContext(ctx, db, "migrations")
}

func OpenPool(ctx context.Context, databaseURL string, settings PoolSettings) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	settings.apply(cfg)
	attachQueryTracer(cfg.ConnConfig)
	return pgxpool.NewWithConfig(ctx, cfg)
}

func Ping(ctx context.Context, db *sql.DB) error {
	return db.PingContext(ctx)
}
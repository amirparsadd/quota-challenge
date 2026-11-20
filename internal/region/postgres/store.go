package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"quota-challenge/internal/region/port"
)

var _ port.QuotaRepository = (*Store)(nil)

type Store struct {
	pool     *pgxpool.Pool
	regionID string
}

func NewStore(pool *pgxpool.Pool, regionID string) *Store {
	return &Store{pool: pool, regionID: regionID}
}

func (s *Store) Pool() *pgxpool.Pool { return s.pool }

func (s *Store) RegionID() string { return s.regionID }

func (s *Store) EnsureRegionMeta(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO region_meta (region_id) VALUES ($1) ON CONFLICT DO NOTHING`, s.regionID)
	return err
}

func (s *Store) AssertRegion(ctx context.Context) error {
	var got string
	err := s.pool.QueryRow(ctx, `SELECT region_id FROM region_meta LIMIT 1`).Scan(&got)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("region_meta missing")
	}
	if err != nil {
		return err
	}
	if got != s.regionID {
		return fmt.Errorf("REGION_ID %q != DB region %q", s.regionID, got)
	}
	return nil
}
package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ReapExpired(ctx context.Context, limit int) (int64, time.Duration, error) {
	var n int64
	var maxLag time.Duration
	for i := 0; i < limit; i++ {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return n, maxLag, err
		}
		var id uuid.UUID
		var customerID string
		var expiredAt time.Time
		err = tx.QueryRow(ctx, `
			SELECT id, customer_id, expires_at FROM services
			WHERE state = 'reserved' AND expires_at IS NOT NULL AND expires_at <= now()
			ORDER BY expires_at
			LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&id, &customerID, &expiredAt)
		if errors.Is(err, pgx.ErrNoRows) {
			_ = tx.Rollback(ctx)
			break
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return n, maxLag, err
		}
		if lag := time.Since(expiredAt); lag > maxLag {
			maxLag = lag
		}
		if err := s.expireLocked(ctx, tx, customerID, id); err != nil {
			_ = tx.Rollback(ctx)
			continue
		}
		if err := tx.Commit(ctx); err != nil {
			continue
		}
		n++
	}
	return n, maxLag, nil
}

func (s *Store) ReclaimExpiredInScope(ctx context.Context, rtIDs []int) error {
	if len(rtIDs) == 0 {
		return nil
	}
	_, _, _ = s.ReapExpired(ctx, 50)
	return nil
}
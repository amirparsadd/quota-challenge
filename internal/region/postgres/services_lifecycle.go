package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"quota-challenge/internal/capacity"
)

func (s *Store) ConfirmService(ctx context.Context, customerID string, id uuid.UUID) (*capacity.Service, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var state string
	var expires *time.Time
	err = tx.QueryRow(ctx, `
		SELECT state, expires_at FROM services WHERE id = $1 AND customer_id = $2 FOR UPDATE`,
		id, customerID).Scan(&state, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, capacity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if state == string(capacity.StateActive) {
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return s.GetService(ctx, customerID, id)
	}
	if state != string(capacity.StateReserved) {
		return nil, capacity.ErrInvalidState
	}
	if expires != nil && time.Now().UTC().After(*expires) {
		if err := s.expireLocked(ctx, tx, customerID, id); err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return nil, capacity.ErrReservationExpired
	}

	lines, err := s.loadLines(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	for _, ln := range lines {
		_, err = tx.Exec(ctx, `
			UPDATE pools SET reserved = reserved - $2, allocated = allocated + $2
			WHERE resource_type_id = $1`, ln.RTID, ln.Amount)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(ctx, `
			UPDATE customer_quotas SET reserved = reserved - $2, allocated = allocated + $2
			WHERE customer_id = $1 AND resource_type_id = $3`,
			customerID, ln.Amount, ln.RTID)
		if err != nil {
			return nil, err
		}
	}
	_, err = tx.Exec(ctx, `
		UPDATE services SET state = 'active', expires_at = NULL, updated_at = now()
		WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetService(ctx, customerID, id)
}

func (s *Store) CancelService(ctx context.Context, customerID string, id uuid.UUID) (*capacity.Service, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var state string
	err = tx.QueryRow(ctx, `
		SELECT state FROM services WHERE id = $1 AND customer_id = $2 FOR UPDATE`,
		id, customerID).Scan(&state)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, capacity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if state == string(capacity.StateCancelled) {
		return txCommitService(ctx, tx, s, customerID, id)
	}
	if state != string(capacity.StateReserved) {
		return nil, capacity.ErrInvalidState
	}
	if err := s.releaseReserved(ctx, tx, customerID, id); err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE services SET state = 'cancelled', updated_at = now() WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	return txCommitService(ctx, tx, s, customerID, id)
}

func (s *Store) ReleaseService(ctx context.Context, id uuid.UUID) (*capacity.Service, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var state, customerID string
	err = tx.QueryRow(ctx, `
		SELECT state, customer_id FROM services WHERE id = $1 FOR UPDATE`, id).
		Scan(&state, &customerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, capacity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if state == string(capacity.StateReleased) {
		return txCommitService(ctx, tx, s, customerID, id)
	}
	if state != string(capacity.StateActive) {
		return nil, capacity.ErrInvalidState
	}
	lines, err := s.loadLines(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	for _, ln := range lines {
		_, err = tx.Exec(ctx, `
			UPDATE pools SET allocated = allocated - $2
			WHERE resource_type_id = $1 AND allocated >= $2`, ln.RTID, ln.Amount)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(ctx, `
			UPDATE customer_quotas SET allocated = allocated - $2
			WHERE customer_id = $1 AND resource_type_id = $3 AND allocated >= $2`,
			customerID, ln.Amount, ln.RTID)
		if err != nil {
			return nil, err
		}
	}
	_, err = tx.Exec(ctx, `
		UPDATE services SET state = 'released', updated_at = now() WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	return txCommitService(ctx, tx, s, customerID, id)
}
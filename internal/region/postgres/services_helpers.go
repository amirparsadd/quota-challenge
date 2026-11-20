package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"quota-challenge/internal/capacity"
)

type lineRow struct {
	RTID   int
	Amount int64
}

func (s *Store) loadLines(ctx context.Context, q pgx.Tx, id uuid.UUID) ([]lineRow, error) {
	rows, err := q.Query(ctx, `
		SELECT resource_type_id, amount FROM service_resources WHERE service_id = $1 ORDER BY resource_type_id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []lineRow
	for rows.Next() {
		var ln lineRow
		if err := rows.Scan(&ln.RTID, &ln.Amount); err != nil {
			return nil, err
		}
		out = append(out, ln)
	}
	return out, nil
}

func (s *Store) releaseReserved(ctx context.Context, tx pgx.Tx, customerID string, id uuid.UUID) error {
	lines, err := s.loadLines(ctx, tx, id)
	if err != nil {
		return err
	}
	for _, ln := range lines {
		_, err = tx.Exec(ctx, `
			UPDATE pools SET reserved = reserved - $2
			WHERE resource_type_id = $1 AND reserved >= $2`, ln.RTID, ln.Amount)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE customer_quotas SET reserved = reserved - $2
			WHERE customer_id = $1 AND resource_type_id = $3 AND reserved >= $2`,
			customerID, ln.Amount, ln.RTID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) expireLocked(ctx context.Context, tx pgx.Tx, customerID string, id uuid.UUID) error {
	if err := s.releaseReserved(ctx, tx, customerID, id); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		UPDATE services SET state = 'expired', updated_at = now() WHERE id = $1`, id)
	return err
}

func txCommitService(ctx context.Context, tx pgx.Tx, s *Store, customerID string, id uuid.UUID) (*capacity.Service, error) {
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetService(ctx, customerID, id)
}
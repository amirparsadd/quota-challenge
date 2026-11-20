package postgres

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// RecordRejectedService persists a durable rejected service row (design lifecycle).
func (s *Store) RecordRejectedService(ctx context.Context, customerID string, productID int, reason string) error {
	id := uuid.New()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO services (id, customer_id, product_id, state, expires_at)
		VALUES ($1, $2, $3, 'rejected', NULL)`, id, customerID, productID)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]string{"reason": reason})
	_, err = tx.Exec(ctx, `
		INSERT INTO service_events (service_id, event_type, payload)
		VALUES ($1, 'rejected', $2)`, id, payload)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
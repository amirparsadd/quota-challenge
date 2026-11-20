package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"quota-challenge/internal/capacity"
)

func (s *Store) ReserveCreate(ctx context.Context, customerID string, productID int, productName string, lines []capacity.ReserveLine, ttl int64, token uuid.UUID, idemKey string) (*capacity.Service, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, ln := range lines {
		tag, err := tx.Exec(ctx, `
			UPDATE customer_quotas
			SET reserved = reserved + $3
			WHERE customer_id = $1 AND resource_type_id = $2
			  AND quota_limit - allocated - reserved >= $3`,
			customerID, ln.ResourceTypeID, ln.Amount)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() == 0 {
			return nil, capacity.ErrQuotaExceeded
		}
	}

	svcID := uuid.New()
	state := capacity.StateReserved
	var expiresAt *time.Time
	if ttl == 0 {
		state = capacity.StateActive
	} else {
		t := time.Now().UTC().Add(time.Duration(ttl) * time.Second)
		expiresAt = &t
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO services (id, customer_id, product_id, state, expires_at)
		VALUES ($1, $2, $3, $4, $5)`, svcID, customerID, productID, state, expiresAt)
	if err != nil {
		return nil, err
	}

	for _, ln := range lines {
		_, err = tx.Exec(ctx, `
			INSERT INTO service_resources (service_id, resource_type_id, amount)
			VALUES ($1, $2, $3)`, svcID, ln.ResourceTypeID, ln.Amount)
		if err != nil {
			return nil, err
		}
	}

	ev := map[string]any{"event": "created", "state": state}
	evb, _ := json.Marshal(ev)
	_, err = tx.Exec(ctx, `
		INSERT INTO service_events (service_id, event_type, payload)
		VALUES ($1, 'created', $2)`, svcID, evb)
	if err != nil {
		return nil, err
	}

	for _, ln := range lines {
		tag, err := tx.Exec(ctx, `
			UPDATE pools SET reserved = reserved + $2
			WHERE resource_type_id = $1
			  AND total - allocated - reserved >= $2`, ln.ResourceTypeID, ln.Amount)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() == 0 {
			return nil, capacity.ErrCapacityExceeded
		}
	}

	if state == capacity.StateActive {
		for _, ln := range lines {
			_, err = tx.Exec(ctx, `
				UPDATE pools SET reserved = reserved - $2, allocated = allocated + $2
				WHERE resource_type_id = $1`, ln.ResourceTypeID, ln.Amount)
			if err != nil {
				return nil, err
			}
			_, err = tx.Exec(ctx, `
				UPDATE customer_quotas SET reserved = reserved - $2, allocated = allocated + $2
				WHERE customer_id = $1 AND resource_type_id = $3`,
				customerID, ln.Amount, ln.ResourceTypeID)
			if err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetService(ctx, customerID, svcID)
}
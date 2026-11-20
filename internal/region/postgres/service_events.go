package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"quota-challenge/internal/capacity"
)

func (s *Store) ListServiceEvents(ctx context.Context, customerID string, serviceID uuid.UUID) ([]capacity.ServiceEvent, error) {
	var owner string
	err := s.pool.QueryRow(ctx, `SELECT customer_id FROM services WHERE id = $1`, serviceID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, capacity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if owner != customerID {
		return nil, capacity.ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, event_type, created_at
		FROM service_events
		WHERE service_id = $1
		ORDER BY id ASC`, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []capacity.ServiceEvent
	for rows.Next() {
		var ev capacity.ServiceEvent
		if err := rows.Scan(&ev.ID, &ev.EventType, &ev.At); err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}
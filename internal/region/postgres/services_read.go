package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"quota-challenge/internal/capacity"
)

func (s *Store) GetService(ctx context.Context, customerID string, id uuid.UUID) (*capacity.Service, error) {
	var svc capacity.Service
	var state string
	var expires *time.Time
	var productName string
	err := s.pool.QueryRow(ctx, `
		SELECT s.id, s.customer_id, p.name, s.state, s.expires_at, s.created_at, s.updated_at
		FROM services s JOIN products p ON p.id = s.product_id
		WHERE s.id = $1 AND s.customer_id = $2`, id, customerID).Scan(
		&svc.ID, &svc.CustomerID, &productName, &state, &expires, &svc.CreatedAt, &svc.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, capacity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	svc.Product = productName
	svc.State = capacity.ServiceState(state)
	svc.ExpiresAt = expires

	rows, err := s.pool.Query(ctx, `
		SELECT rt.name, rt.unit, sr.amount FROM service_resources sr
		JOIN resource_types rt ON rt.id = sr.resource_type_id
		WHERE sr.service_id = $1 ORDER BY rt.name`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var ln capacity.ResourceLine
		if err := rows.Scan(&ln.Resource, &ln.Unit, &ln.Amount); err != nil {
			return nil, err
		}
		svc.Items = append(svc.Items, ln)
	}
	return &svc, nil
}

func (s *Store) ListServices(ctx context.Context, customerID string) ([]capacity.Service, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id FROM services WHERE customer_id = $1 ORDER BY created_at DESC`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []capacity.Service
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		svc, err := s.GetService(ctx, customerID, id)
		if err != nil {
			return nil, err
		}
		out = append(out, *svc)
	}
	return out, nil
}
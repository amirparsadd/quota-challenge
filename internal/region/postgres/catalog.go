package postgres

import (
	"context"

	"quota-challenge/internal/capacity"
)

func (s *Store) ResourceByName(ctx context.Context, name string) (capacity.ResourceType, error) {
	var rt capacity.ResourceType
	var max *int64
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, unit, min_amount, step_amount, max_amount, discrete
		FROM resource_types WHERE name = $1`, name).Scan(
		&rt.ID, &rt.Name, &rt.Unit, &rt.MinAmount, &rt.StepAmount, &max, &rt.Discrete)
	rt.MaxAmount = max
	return rt, err
}

func (s *Store) ProductID(ctx context.Context, product string) (id, defaultTTL int, err error) {
	id, defaultTTL, _, err = s.ProductTTLs(ctx, product)
	return id, defaultTTL, err
}

func (s *Store) ProductTTLs(ctx context.Context, product string) (id, defaultTTL, maxTTL int, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT id, default_ttl_seconds, max_ttl_seconds FROM products WHERE name = $1`, product).
		Scan(&id, &defaultTTL, &maxTTL)
	return id, defaultTTL, maxTTL, err
}
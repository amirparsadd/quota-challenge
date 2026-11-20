package postgres

import (
	"context"

	"quota-challenge/internal/capacity"
)

// ListProducts returns catalog products and their resource constraints for the region.
func (s *Store) ListProducts(ctx context.Context) ([]capacity.Product, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.name, p.default_ttl_seconds, p.max_ttl_seconds,
		       rt.name, rt.unit, rt.min_amount, rt.step_amount
		FROM products p
		JOIN product_resources pr ON pr.product_id = p.id
		JOIN resource_types rt ON rt.id = pr.resource_type_id
		ORDER BY p.name, rt.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byName := make(map[string]*capacity.Product)
	var order []string
	for rows.Next() {
		var (
			pname              string
			defaultTTL, maxTTL int
			res, unit          string
			minAmt, step       int64
		)
		if err := rows.Scan(&pname, &defaultTTL, &maxTTL, &res, &unit, &minAmt, &step); err != nil {
			return nil, err
		}
		prod, ok := byName[pname]
		if !ok {
			prod = &capacity.Product{
				Name: pname, DefaultTTLSeconds: defaultTTL, MaxTTLSeconds: maxTTL,
				Resources: []capacity.ProductResource{},
			}
			byName[pname] = prod
			order = append(order, pname)
		}
		prod.Resources = append(prod.Resources, capacity.ProductResource{
			Resource: res, Unit: unit, MinAmount: minAmt, StepAmount: step,
		})
	}
	out := make([]capacity.Product, 0, len(order))
	for _, name := range order {
		out = append(out, *byName[name])
	}
	return out, rows.Err()
}
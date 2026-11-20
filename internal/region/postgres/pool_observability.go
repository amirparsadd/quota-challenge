package postgres

import (
	"context"

	"quota-challenge/internal/telemetry"
)

// PoolUtilizationSamples returns per-resource pool utilization for metrics.
func (s *Store) PoolUtilizationSamples(ctx context.Context) []telemetry.PoolUtilSample {
	rows, err := s.pool.Query(ctx, `
		SELECT rt.name, p.total, p.allocated, p.reserved
		FROM pools p
		JOIN resource_types rt ON rt.id = p.resource_type_id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []telemetry.PoolUtilSample
	for rows.Next() {
		var name string
		var total, alloc, res int64
		if err := rows.Scan(&name, &total, &alloc, &res); err != nil {
			return out
		}
		out = append(out, telemetry.MakePoolUtilSample(name, telemetry.PoolUtilFromTotals(total, alloc, res)))
	}
	return out
}
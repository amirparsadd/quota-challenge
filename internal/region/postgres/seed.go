package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedDemo inserts ir-thr region metadata, vm product, pools, and demo customer quotas.
func SeedDemo(ctx context.Context, pool *pgxpool.Pool, regionID string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO region_meta (region_id) VALUES ($1)
		ON CONFLICT (region_id) DO NOTHING`, regionID); err != nil {
		return err
	}

	resources := []struct {
		name, unit string
		min, step  int64
		discrete   bool
		poolTotal  int64
		demoQuota  int64
	}{
		{"cpu", "cores", 1, 1, false, 1000, 100},
		{"ram", "GiB", 1, 1, false, 2000, 200},
		{"disk", "GiB", 10, 10, false, 50000, 5000},
		{"public_ip", "count", 1, 1, true, 500, 50},
	}

	for _, r := range resources {
		var rtID int
		err := tx.QueryRow(ctx, `
			INSERT INTO resource_types (name, unit, min_amount, step_amount, discrete)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
			RETURNING id`, r.name, r.unit, r.min, r.step, r.discrete).Scan(&rtID)
		if err != nil {
			return fmt.Errorf("resource %s: %w", r.name, err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO pools (resource_type_id, total) VALUES ($1, $2)
			ON CONFLICT (resource_type_id) DO UPDATE SET total = EXCLUDED.total`,
			rtID, r.poolTotal)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO customer_quotas (customer_id, resource_type_id, quota_limit)
			VALUES ('demo', $1, $2)
			ON CONFLICT (customer_id, resource_type_id) DO UPDATE SET quota_limit = EXCLUDED.quota_limit`,
			rtID, r.demoQuota)
		if err != nil {
			return err
		}
	}

	var productID int
	err = tx.QueryRow(ctx, `
		INSERT INTO products (name) VALUES ('vm')
		ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id`).Scan(&productID)
	if err != nil {
		return err
	}

	rows, err := tx.Query(ctx, `SELECT id, name FROM resource_types`)
	if err != nil {
		return err
	}
	type rtRow struct {
		id int
	}
	var rts []rtRow
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			rows.Close()
			return err
		}
		rts = append(rts, rtRow{id: id})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, rt := range rts {
		_, err = tx.Exec(ctx, `
			INSERT INTO product_resources (product_id, resource_type_id)
			VALUES ($1, $2) ON CONFLICT DO NOTHING`, productID, rt.id)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
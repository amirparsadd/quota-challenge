package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"quota-challenge/internal/capacity"
)

func (s *Store) GetAvailability(ctx context.Context, customerID, product string) ([]capacity.AvailabilityEntry, error) {
	qry := `
		SELECT rt.name, rt.unit,
		       p.total - p.allocated - p.reserved AS pool_rem,
		       COALESCE(cq.quota_limit, 0) - COALESCE(cq.allocated, 0) - COALESCE(cq.reserved, 0) AS quota_rem
		FROM pools p
		JOIN resource_types rt ON rt.id = p.resource_type_id
		LEFT JOIN customer_quotas cq ON cq.resource_type_id = rt.id AND cq.customer_id = $1`
	args := []any{customerID}
	if product != "" {
		qry += `
		JOIN product_resources pr ON pr.resource_type_id = rt.id
		JOIN products prod ON prod.id = pr.product_id AND prod.name = $2`
		args = append(args, product)
	}
	qry += ` ORDER BY rt.name`
	rows, err := s.pool.Query(ctx, qry, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []capacity.AvailabilityEntry
	for rows.Next() {
		var e capacity.AvailabilityEntry
		if err := rows.Scan(&e.Resource, &e.Unit, &e.PoolRemaining, &e.QuotaRemaining); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func (s *Store) GetQuota(ctx context.Context, customerID string) ([]capacity.QuotaEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT rt.name, rt.unit, cq.quota_limit, COALESCE(cq.allocated, 0), COALESCE(cq.reserved, 0),
		       (cq.customer_id IS NOT NULL) AS provisioned
		FROM resource_types rt
		LEFT JOIN customer_quotas cq ON cq.resource_type_id = rt.id AND cq.customer_id = $1
		ORDER BY rt.name`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []capacity.QuotaEntry
	for rows.Next() {
		var e capacity.QuotaEntry
		var limit *int64
		if err := rows.Scan(&e.Resource, &e.Unit, &limit, &e.Allocated, &e.Reserved, &e.Provisioned); err != nil {
			return nil, err
		}
		if e.Provisioned && limit != nil {
			e.Limit = *limit
		}
		out = append(out, e)
	}
	return out, nil
}

func (s *Store) SetPoolTotal(ctx context.Context, resource string, total int64) (*capacity.Pool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE pools SET total = $2
		FROM resource_types rt WHERE rt.id = pools.resource_type_id AND rt.name = $1`,
		resource, total)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, capacity.ErrNotFound
	}
	return s.GetPool(ctx, resource)
}

func (s *Store) SetCustomerQuota(ctx context.Context, customerID, resource string, limit int64) error {
	rt, err := s.ResourceByName(ctx, resource)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO customer_quotas (customer_id, resource_type_id, quota_limit)
		VALUES ($1, $2, $3)
		ON CONFLICT (customer_id, resource_type_id) DO UPDATE SET quota_limit = EXCLUDED.quota_limit`,
		customerID, rt.ID, limit)
	return err
}

func (s *Store) GetPool(ctx context.Context, resource string) (*capacity.Pool, error) {
	var p capacity.Pool
	err := s.pool.QueryRow(ctx, `
		SELECT rt.name, p.total, p.baseline_allocated, p.allocated, p.reserved
		FROM pools p JOIN resource_types rt ON rt.id = p.resource_type_id
		WHERE rt.name = $1`, resource).Scan(&p.Resource, &p.Total, &p.BaselineAllocated, &p.Allocated, &p.Reserved)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, capacity.ErrNotFound
	}
	return &p, err
}

func (s *Store) ResyncCounters(ctx context.Context, apply bool) (int64, error) {
	var drift int64
	rows, err := s.pool.Query(ctx, `
		SELECT p.resource_type_id,
		       p.allocated, p.reserved,
		       COALESCE(SUM(CASE WHEN s.state = 'active' THEN sr.amount ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN s.state = 'reserved' THEN sr.amount ELSE 0 END), 0)
		FROM pools p
		LEFT JOIN service_resources sr ON sr.resource_type_id = p.resource_type_id
		LEFT JOIN services s ON s.id = sr.service_id
		GROUP BY p.resource_type_id, p.allocated, p.reserved`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var rtID int
		var alloc, res, la, lr int64
		if err := rows.Scan(&rtID, &alloc, &res, &la, &lr); err != nil {
			return 0, err
		}
		drift += absInt64(alloc-la) + absInt64(res-lr)
		if apply {
			_, err = s.pool.Exec(ctx, `UPDATE pools SET allocated = $2, reserved = $3 WHERE resource_type_id = $1`,
				rtID, la, lr)
			if err != nil {
				return drift, err
			}
		}
	}
	cqRows, err := s.pool.Query(ctx, `
		SELECT cq.customer_id, cq.resource_type_id, cq.allocated, cq.reserved,
		       COALESCE(SUM(CASE WHEN s.state = 'active' THEN sr.amount ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN s.state = 'reserved' THEN sr.amount ELSE 0 END), 0)
		FROM customer_quotas cq
		LEFT JOIN service_resources sr ON sr.resource_type_id = cq.resource_type_id
		LEFT JOIN services s ON s.id = sr.service_id AND s.customer_id = cq.customer_id
		GROUP BY cq.customer_id, cq.resource_type_id, cq.allocated, cq.reserved`)
	if err != nil {
		return drift, err
	}
	defer cqRows.Close()
	for cqRows.Next() {
		var cust string
		var rtID int
		var alloc, res, la, lr int64
		if err := cqRows.Scan(&cust, &rtID, &alloc, &res, &la, &lr); err != nil {
			return drift, err
		}
		drift += absInt64(alloc-la) + absInt64(res-lr)
		if apply {
			_, err = s.pool.Exec(ctx, `
				UPDATE customer_quotas SET allocated = $3, reserved = $4
				WHERE customer_id = $1 AND resource_type_id = $2`,
				cust, rtID, la, lr)
			if err != nil {
				return drift, err
			}
		}
	}
	_, _ = s.pool.Exec(ctx, `INSERT INTO counter_resyncs (dry_run, drift) VALUES ($1, $2)`, !apply, drift)
	return drift, nil
}

func absInt64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
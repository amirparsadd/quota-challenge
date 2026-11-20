//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"quota-challenge/internal/capacity"
	pstore "quota-challenge/internal/region/postgres"
)

func TestResyncDryRunThenApplyZeroDrift(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	_, _, err := svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm", Items: vmItems(),
		TTLSeconds: capacity.TTLDefaultUnset,
	})
	require.NoError(t, err)
	dry, err := svc.Resync(ctx, false)
	require.NoError(t, err)
	require.Equal(t, int64(0), dry)
	got, err := svc.Resync(ctx, true)
	require.NoError(t, err)
	require.Equal(t, int64(0), got)
}

func TestIdempotencyPayloadMismatch(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	key := uuid.NewString()
	_, _, err := svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: key, Product: "vm", Items: vmItems(),
		TTLSeconds: capacity.TTLDefaultUnset,
	})
	require.NoError(t, err)
	_, _, err = svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: key, Product: "vm",
		Items: []capacity.ResourceLine{{Resource: "cpu", Amount: 4}},
		TTLSeconds: capacity.TTLDefaultUnset,
	})
	require.ErrorIs(t, err, capacity.ErrIdempotencyMismatch)
}

func TestOnDemandReclaimFreesCapacity(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	pool, err := pstore.OpenPool(ctx, dsn, pstore.PoolSettings{})
	require.NoError(t, err)
	defer pool.Close()
	store := pstore.NewStore(pool, "ir-thr-1")

	_, err = store.SetPoolTotal(ctx, "public_ip", 1)
	require.NoError(t, err)
	err = store.SetCustomerQuota(ctx, "demo", "public_ip", 10)
	require.NoError(t, err)

	hold, _, err := svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm",
		Items: []capacity.ResourceLine{
			{Resource: "cpu", Amount: 2}, {Resource: "ram", Amount: 4},
			{Resource: "disk", Amount: 40}, {Resource: "public_ip", Amount: 1},
		},
		TTLSeconds: 1,
	})
	require.NoError(t, err)

	_, _, err = svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm",
		Items: []capacity.ResourceLine{
			{Resource: "cpu", Amount: 2}, {Resource: "ram", Amount: 4},
			{Resource: "disk", Amount: 40}, {Resource: "public_ip", Amount: 1},
		},
		TTLSeconds: capacity.TTLDefaultUnset,
	})
	require.Error(t, err)

	time.Sleep(1200 * time.Millisecond)

	_, _, err = svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm",
		Items: []capacity.ResourceLine{
			{Resource: "cpu", Amount: 2}, {Resource: "ram", Amount: 4},
			{Resource: "disk", Amount: 40}, {Resource: "public_ip", Amount: 1},
		},
		TTLSeconds: capacity.TTLDefaultUnset,
	})
	require.NoError(t, err)
	_ = hold
}

func TestDoubleReapSafe(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	_, _, err := svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm",
		Items: vmItems(), TTLSeconds: 1,
	})
	require.NoError(t, err)
	time.Sleep(1200 * time.Millisecond)
	n1, err := svc.Reap(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, n1, int64(1))
	n2, err := svc.Reap(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(0), n2)
}
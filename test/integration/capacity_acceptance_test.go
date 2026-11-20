//go:build integration

package integration

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"quota-challenge/internal/capacity"
	pstore "quota-challenge/internal/region/postgres"
)

func TestPublicIPPoolExactlyOneWinner(t *testing.T) {
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

	const workers = 15
	var wg sync.WaitGroup
	var okCount atomic.Int32
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := svc.Reserve(ctx, capacity.ReserveRequest{
				CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm",
				Items: []capacity.ResourceLine{
					{Resource: "cpu", Amount: 2}, {Resource: "ram", Amount: 4},
					{Resource: "disk", Amount: 40}, {Resource: "public_ip", Amount: 1},
				},
				TTLSeconds: capacity.TTLDefaultUnset,
			})
			if err == nil {
				okCount.Add(1)
			}
		}()
	}
	wg.Wait()
	require.Equal(t, int32(1), okCount.Load())
}

func TestQuotaCapExactSuccesses(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	pool, err := pstore.OpenPool(ctx, dsn, pstore.PoolSettings{})
	require.NoError(t, err)
	defer pool.Close()
	store := pstore.NewStore(pool, "ir-thr-1")
	require.NoError(t, store.SetCustomerQuota(ctx, "demo", "cpu", 4))

	const workers = 10
	var wg sync.WaitGroup
	var okCount atomic.Int32
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := svc.Reserve(ctx, capacity.ReserveRequest{
				CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm",
				Items: vmItems(), TTLSeconds: capacity.TTLDefaultUnset,
			})
			if err == nil {
				okCount.Add(1)
			}
		}()
	}
	wg.Wait()
	require.Equal(t, int32(2), okCount.Load())
}

func TestLoweredQuotaBlocksNewCreate(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	pool, err := pstore.OpenPool(ctx, dsn, pstore.PoolSettings{})
	require.NoError(t, err)
	defer pool.Close()
	store := pstore.NewStore(pool, "ir-thr-1")

	_, _, err = svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm",
		Items: vmItems(), TTLSeconds: 0,
	})
	require.NoError(t, err)
	require.NoError(t, store.SetCustomerQuota(ctx, "demo", "cpu", 2))

	_, _, err = svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm",
		Items: vmItems(), TTLSeconds: capacity.TTLDefaultUnset,
	})
	require.Error(t, err)
}

func TestMultiLineCreateRollsBack(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	pool, err := pstore.OpenPool(ctx, dsn, pstore.PoolSettings{})
	require.NoError(t, err)
	defer pool.Close()
	store := pstore.NewStore(pool, "ir-thr-1")
	_, err = store.SetPoolTotal(ctx, "disk", 30)
	require.NoError(t, err)

	before, err := store.GetPool(ctx, "cpu")
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

	after, err := store.GetPool(ctx, "cpu")
	require.NoError(t, err)
	require.Equal(t, before.Reserved, after.Reserved)
	require.Equal(t, before.Allocated, after.Allocated)

	list, err := svc.List(ctx, "demo")
	require.NoError(t, err)
	for _, s := range list {
		require.NotEqual(t, capacity.StateReserved, s.State)
	}
}

func TestConcurrentIdempotencySameKey(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	key := uuid.NewString()
	req := capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: key, Product: "vm", Items: vmItems(),
		TTLSeconds: capacity.TTLDefaultUnset,
	}
	const workers = 20
	var wg sync.WaitGroup
	ids := make([]uuid.UUID, workers)
	errs := make([]error, workers)
	for i := range workers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s, _, err := svc.Reserve(ctx, req)
			if err == nil {
				ids[i] = s.ID
			}
			errs[i] = err
		}(i)
	}
	wg.Wait()
	var successID uuid.UUID
	successes := 0
	for i := range workers {
		if errs[i] == nil {
			successes++
			if successID == uuid.Nil {
				successID = ids[i]
			} else {
				require.Equal(t, successID, ids[i])
			}
		} else {
			require.ErrorIs(t, errs[i], capacity.ErrIdempotencyConflict)
		}
	}
	require.GreaterOrEqual(t, successes, 1)
}

func TestAvailabilityProductFilter(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	all, err := svc.Availability(ctx, "demo", "")
	require.NoError(t, err)
	vmOnly, err := svc.Availability(ctx, "demo", "vm")
	require.NoError(t, err)
	require.Len(t, vmOnly, 4)
	require.GreaterOrEqual(t, len(all), len(vmOnly))
}

func TestResyncUnderConcurrentCreates(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = svc.Reserve(ctx, capacity.ReserveRequest{
				CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm",
				Items: vmItems(), TTLSeconds: capacity.TTLDefaultUnset,
			})
		}()
	}
	wg.Wait()
	drift, err := svc.Resync(ctx, false)
	require.NoError(t, err)
	require.Equal(t, int64(0), drift)
}
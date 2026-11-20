//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"quota-challenge/internal/capacity"
	pstore "quota-challenge/internal/region/postgres"
	"quota-challenge/internal/region/regionsvc"
)

func startPostgres(t *testing.T) (string, func()) {
	t.Helper()
	ctx := context.Background()
	pgTag := os.Getenv("POSTGRES_VERSION")
	if pgTag == "" {
		pgTag = "16.16-bookworm" // bare `go test`; `make test` exports versions.env
	}
	ctr, err := postgres.Run(ctx,
		fmt.Sprintf("postgres:%s", pgTag),
		postgres.WithDatabase("quota"),
		postgres.WithUsername("quota"),
		postgres.WithPassword("dev"),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	waitPostgresReady(t, dsn)
	return dsn, func() { _ = testcontainers.TerminateContainer(ctr) }
}

func waitPostgresReady(t *testing.T, dsn string) {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		pool, err := pstore.OpenPool(ctx, dsn, pstore.PoolSettings{})
		if err != nil {
			lastErr = err
			time.Sleep(200 * time.Millisecond)
			continue
		}
		err = pool.Ping(ctx)
		pool.Close()
		if err == nil {
			return
		}
		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}
	require.NoError(t, lastErr, "postgres not ready for connections")
}

func setupRegion(t *testing.T, dsn string) *regionsvc.Service {
	return setupRegionID(t, dsn, "ir-thr-1")
}

func setupRegionID(t *testing.T, dsn, regionID string) *regionsvc.Service {
	svc, _ := setupRegionIDWithRepo(t, dsn, regionID)
	return svc
}

func setupRegionIDWithRepo(t *testing.T, dsn, regionID string) (*regionsvc.Service, *pstore.Store) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, pstore.Migrate(ctx, dsn))
	pool, err := pstore.OpenPool(ctx, dsn, pstore.PoolSettings{})
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })
	require.NoError(t, pstore.SeedDemo(ctx, pool, regionID))
	store := pstore.NewStore(pool, regionID)
	require.NoError(t, store.AssertRegion(ctx))
	return regionsvc.New(store), store
}

func vmItems() []capacity.ResourceLine {
	return []capacity.ResourceLine{
		{Resource: "cpu", Amount: 2},
		{Resource: "ram", Amount: 4},
		{Resource: "disk", Amount: 40},
		{Resource: "public_ip", Amount: 1},
	}
}

func TestReserveIdempotencyReplay(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	key := uuid.NewString()
	req := capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: key, Product: "vm", Items: vmItems(), TTLSeconds: capacity.TTLDefaultUnset,
	}
	s1, replay1, err := svc.Reserve(ctx, req)
	require.NoError(t, err)
	require.False(t, replay1)
	s2, replay2, err := svc.Reserve(ctx, req)
	require.NoError(t, err)
	require.True(t, replay2)
	require.Equal(t, s1.ID, s2.ID)
}

func TestConfirmAndCancel(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	s, _, err := svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm", Items: vmItems(), TTLSeconds: capacity.TTLDefaultUnset,
	})
	require.NoError(t, err)
	confirmed, err := svc.Confirm(ctx, "demo", s.ID.String())
	require.NoError(t, err)
	require.Equal(t, capacity.StateActive, confirmed.State)

	s2, _, err := svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm", Items: vmItems(), TTLSeconds: capacity.TTLDefaultUnset,
	})
	require.NoError(t, err)
	cancelled, err := svc.Cancel(ctx, "demo", s2.ID.String())
	require.NoError(t, err)
	require.Equal(t, capacity.StateCancelled, cancelled.State)
}

func TestRejectedCreatesServiceRow(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	_, _, err := svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm",
		Items: []capacity.ResourceLine{{Resource: "cpu", Amount: 999999}},
		TTLSeconds: capacity.TTLDefaultUnset,
	})
	require.Error(t, err)
	list, err := svc.List(ctx, "demo")
	require.NoError(t, err)
	var rejected int
	for _, s := range list {
		if s.State == capacity.StateRejected {
			rejected++
		}
	}
	require.GreaterOrEqual(t, rejected, 1)
}

func TestRejectedIdempotencyReplay(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	key := uuid.NewString()
	big := []capacity.ResourceLine{{Resource: "cpu", Amount: 999999}}
	_, _, err := svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: key, Product: "vm", Items: big,
	})
	require.Error(t, err)
	_, replay, err2 := svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: key, Product: "vm", Items: big,
	})
	require.True(t, replay)
	var replayHTTP *capacity.ReplayHTTPError
	require.ErrorAs(t, err2, &replayHTTP)
	require.Equal(t, 409, replayHTTP.Status)
	require.Contains(t, string(replayHTTP.Body), "/v1/problems/")
}

func TestExpiryReap(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	ttl := int64(1)
	s, _, err := svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm", Items: vmItems(), TTLSeconds: ttl,
	})
	require.NoError(t, err)
	time.Sleep(1500 * time.Millisecond)
	n, err := svc.Reap(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, n, int64(1))
	got, err := svc.Get(ctx, "demo", s.ID.String())
	require.NoError(t, err)
	require.Equal(t, capacity.StateExpired, got.State)
}
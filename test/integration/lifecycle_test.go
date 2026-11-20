//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"quota-challenge/internal/capacity"
)

func TestTTLZeroCreatesActive(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	s, _, err := svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm",
		Items: vmItems(), TTLSeconds: 0,
	})
	require.NoError(t, err)
	require.Equal(t, capacity.StateActive, s.State)
	require.Nil(t, s.ExpiresAt)
}

func TestReleaseActiveService(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	s, _, err := svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm",
		Items: vmItems(), TTLSeconds: 0,
	})
	require.NoError(t, err)
	out, err := svc.Release(ctx, s.ID.String())
	require.NoError(t, err)
	require.Equal(t, capacity.StateReleased, out.State)
}

func TestCancelIsIdempotent(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	s, _, err := svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm", Items: vmItems(),
		TTLSeconds: capacity.TTLDefaultUnset,
	})
	require.NoError(t, err)
	c1, err := svc.Cancel(ctx, "demo", s.ID.String())
	require.NoError(t, err)
	c2, err := svc.Cancel(ctx, "demo", s.ID.String())
	require.NoError(t, err)
	require.Equal(t, capacity.StateCancelled, c1.State)
	require.Equal(t, c1.ID, c2.ID)
}

func TestServiceItemsIncludeUnit(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	s, _, err := svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm",
		Items: vmItems(), TTLSeconds: capacity.TTLDefaultUnset,
	})
	require.NoError(t, err)
	got, err := svc.Get(ctx, "demo", s.ID.String())
	require.NoError(t, err)
	require.NotEmpty(t, got.Items)
	for _, it := range got.Items {
		require.NotEmpty(t, it.Unit, it.Resource)
	}
}

func TestConfirmIsIdempotent(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc := setupRegion(t, dsn)
	ctx := context.Background()
	s, _, err := svc.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm", Items: vmItems(),
		TTLSeconds: capacity.TTLDefaultUnset,
	})
	require.NoError(t, err)
	a1, err := svc.Confirm(ctx, "demo", s.ID.String())
	require.NoError(t, err)
	a2, err := svc.Confirm(ctx, "demo", s.ID.String())
	require.NoError(t, err)
	require.Equal(t, capacity.StateActive, a1.State)
	require.Equal(t, a1.ID, a2.ID)
}
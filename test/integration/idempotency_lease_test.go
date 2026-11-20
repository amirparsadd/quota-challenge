//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"quota-challenge/internal/capacity"
	"quota-challenge/internal/region/regionsvc"
)

func TestIdempotencyLeaseTakeoverAfterExpiry(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	svc, repo := setupRegionIDWithRepo(t, dsn, "ir-thr-1")
	ctx := context.Background()
	key := uuid.NewString()
	req := capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: key, Product: "vm",
		Items: vmItems(), TTLSeconds: capacity.TTLDefaultUnset,
	}
	hash := regionsvc.HashReserve(req.Product, req.Items, req.TTLSeconds)
	token, err := repo.ClaimIdempotency(ctx, req.CustomerID, key, hash, time.Millisecond)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, token)
	time.Sleep(25 * time.Millisecond)

	s, replay, err := svc.Reserve(ctx, req)
	require.NoError(t, err)
	require.False(t, replay)
	require.NotNil(t, s)
}

func TestIdempotencyCompletionFencedOnLeaseToken(t *testing.T) {
	dsn, cleanup := startPostgres(t)
	defer cleanup()
	_, repo := setupRegionIDWithRepo(t, dsn, "ir-thr-1")
	ctx := context.Background()
	key := uuid.NewString()
	hash := regionsvc.HashReserve("vm", vmItems(), capacity.TTLDefaultUnset)
	token, err := repo.ClaimIdempotency(ctx, "demo", key, hash, 15*time.Second)
	require.NoError(t, err)
	wrong := uuid.New()
	err = repo.SaveCompletedReserve(ctx, "demo", key, wrong, &capacity.Service{ID: uuid.New()})
	require.ErrorIs(t, err, capacity.ErrIdempotencyConflict)
	_ = token
}
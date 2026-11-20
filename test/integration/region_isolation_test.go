//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"quota-challenge/internal/capacity"
)

func TestPerRegionDatabaseIsolation(t *testing.T) {
	dsnA, cleanupA := startPostgres(t)
	defer cleanupA()
	dsnB, cleanupB := startPostgres(t)
	defer cleanupB()
	svcA := setupRegionID(t, dsnA, "ir-thr-1")
	svcB := setupRegionID(t, dsnB, "ir-thr-2")
	ctx := context.Background()

	s, _, err := svcA.Reserve(ctx, capacity.ReserveRequest{
		CustomerID: "demo", IdempotencyKey: uuid.NewString(), Product: "vm",
		Items: vmItems(), TTLSeconds: capacity.TTLDefaultUnset,
	})
	require.NoError(t, err)

	listB, err := svcB.List(ctx, "demo")
	require.NoError(t, err)
	for _, x := range listB {
		require.NotEqual(t, s.ID, x.ID)
	}
	_, err = svcB.Get(ctx, "demo", s.ID.String())
	require.ErrorIs(t, err, capacity.ErrNotFound)
}
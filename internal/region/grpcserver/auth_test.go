package grpcserver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestDenyGatewayBlocksAdminCaller(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-caller", callerGateway))
	require.Error(t, denyGateway(ctx))
}

func TestDenyGatewayAllowsOtherCallers(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-caller", "ops-cli"))
	require.NoError(t, denyGateway(ctx))
}
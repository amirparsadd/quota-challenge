package obslog

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// UnaryServerTimeoutInterceptor caps handler work with a deadline when limit > 0.
func UnaryServerTimeoutInterceptor(limit time.Duration) grpc.UnaryServerInterceptor {
	if limit <= 0 {
		return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			return handler(ctx, req)
		}
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if skipGRPCMethod(info.FullMethod) {
			return handler(ctx, req)
		}
		ctx, cancel := context.WithTimeout(ctx, limit)
		defer cancel()
		return handler(ctx, req)
	}
}
package obslog

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func skipGRPCMethod(fullMethod string) bool {
	return strings.Contains(fullMethod, "grpc.health.v1.Health/")
}

// UnaryServerInterceptor logs inbound gRPC calls.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if skipGRPCMethod(info.FullMethod) {
			return handler(ctx, req)
		}
		start := time.Now()
		resp, err := handler(ctx, req)
		logRPC("grpc server", "", info.FullMethod, time.Since(start), err)
		return resp, err
	}
}

// UnaryClientInterceptor logs outbound gRPC calls from the gateway.
func UnaryClientInterceptor(region string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if skipGRPCMethod(method) {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		logRPC("grpc client", region, method, time.Since(start), err)
		return err
	}
}

func logRPC(kind, region, method string, dur time.Duration, err error) {
	attrs := []any{
		"kind", kind,
		"method", method,
		"duration_ms", dur.Milliseconds(),
	}
	if region != "" {
		attrs = append(attrs, "region", region)
	}
	code := codes.OK
	msg := ""
	if err != nil {
		if st, ok := status.FromError(err); ok {
			code = st.Code()
			msg = st.Message()
		} else {
			code = codes.Unknown
			msg = err.Error()
		}
		attrs = append(attrs, "grpc_code", code.String(), "detail", msg)
	} else {
		attrs = append(attrs, "grpc_code", code.String())
	}
	switch {
	case code == codes.OK:
		slog.Debug("rpc call", attrs...)
	case code == codes.Internal || code == codes.Unavailable || code == codes.Unknown:
		slog.Error("rpc call", attrs...)
	default:
		slog.Warn("rpc call", attrs...)
	}
}
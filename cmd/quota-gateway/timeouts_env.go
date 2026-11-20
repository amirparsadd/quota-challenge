package main

import (
	"log/slog"
	"os"
	"time"

	"quota-challenge/internal/gateway"
)

const (
	envHTTPReadHeaderTimeout       = "HTTP_READ_HEADER_TIMEOUT"
	envHTTPShutdownTimeout         = "HTTP_SHUTDOWN_TIMEOUT"
	envGatewayHealthcheckTimeout   = "GATEWAY_HEALTHCHECK_TIMEOUT"
	envGatewayGRPCRegionTimeout    = "GATEWAY_GRPC_REGION_TIMEOUT"
	envGatewayGRPCFanoutTimeout    = "GATEWAY_GRPC_FANOUT_TIMEOUT"
	envGatewayGRPCHealthTimeout    = "GATEWAY_GRPC_HEALTH_TIMEOUT"
	envGatewayCircuitBreakerOpen   = "GATEWAY_CIRCUIT_BREAKER_OPEN_TIMEOUT"
)

func envDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		slog.Warn("invalid duration env, using default", "key", key, "value", v, "default", def)
		return def
	}
	return d
}

func gatewayTimeoutsFromEnv() gateway.Timeouts {
	def := gateway.DefaultTimeouts()
	return gateway.Timeouts{
		RegionRPC:   envDuration(envGatewayGRPCRegionTimeout, 0),
		FanoutRPC:   envDuration(envGatewayGRPCFanoutTimeout, def.FanoutRPC),
		HealthRPC:   envDuration(envGatewayGRPCHealthTimeout, def.HealthRPC),
		BreakerOpen: envDuration(envGatewayCircuitBreakerOpen, def.BreakerOpen),
	}
}

func httpServerTimeoutsFromEnv() (readHeader, shutdown time.Duration) {
	return envDuration(envHTTPReadHeaderTimeout, 5*time.Second),
		envDuration(envHTTPShutdownTimeout, 10*time.Second)
}

func healthcheckTimeoutFromEnv() time.Duration {
	return envDuration(envGatewayHealthcheckTimeout, 3*time.Second)
}
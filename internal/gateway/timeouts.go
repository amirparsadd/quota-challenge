package gateway

import (
	"context"
	"time"
)

// Timeouts tune gateway HTTP→gRPC deadlines and circuit-breaker recovery.
type Timeouts struct {
	// RegionRPC bounds single-region API calls; zero leaves the request context unchanged.
	RegionRPC time.Duration
	FanoutRPC time.Duration
	HealthRPC time.Duration
	BreakerOpen time.Duration
}

func DefaultTimeouts() Timeouts {
	return Timeouts{
		FanoutRPC:   3 * time.Second,
		HealthRPC:   2 * time.Second,
		BreakerOpen: 30 * time.Second,
	}
}

func rpcContext(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		return parent, func() {}
	}
	return context.WithTimeout(parent, d)
}
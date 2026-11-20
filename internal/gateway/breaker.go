package gateway

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"quota-challenge/internal/telemetry"
)

// BreakerState values for metrics.
const (
	BreakerClosed   int64 = 0
	BreakerOpen     int64 = 1
	BreakerHalfOpen int64 = 2
)

// Breaker is a simple per-region circuit breaker.
type Breaker struct {
	region string

	mu              sync.Mutex
	state           int64
	failures        int
	threshold       int
	openFor         time.Duration
	openUntil       time.Time
	halfOpenProbing bool
}

func NewBreaker(region string, openFor time.Duration) *Breaker {
	if openFor <= 0 {
		openFor = 30 * time.Second
	}
	return &Breaker{
		region:    region,
		state:     BreakerClosed,
		threshold: 3,
		openFor:   openFor,
	}
}

func (b *Breaker) State() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.advance()
	return b.state
}

func (b *Breaker) setMetric(ctx context.Context, state int64) {
	if telemetry.Global != nil {
		telemetry.Global.SetBreakerState(ctx, b.region, state)
	}
}

func (b *Breaker) advance() {
	if b.state == BreakerOpen && time.Now().After(b.openUntil) {
		b.state = BreakerHalfOpen
		b.halfOpenProbing = true
	}
}

// BeforeCall returns an error when the circuit is open.
func (b *Breaker) BeforeCall(ctx context.Context) error {
	b.mu.Lock()
	b.advance()
	st := b.state
	b.mu.Unlock()
	b.setMetric(ctx, st)
	if st == BreakerOpen {
		return status.Error(codes.Unavailable, "region circuit breaker open")
	}
	return nil
}

// AfterCall updates breaker state from the RPC result.
func (b *Breaker) AfterCall(ctx context.Context, err error) {
	b.mu.Lock()
	b.advance()
	if err == nil {
		b.failures = 0
		b.state = BreakerClosed
		b.halfOpenProbing = false
	} else if st, ok := status.FromError(err); ok && st.Code() == codes.Unavailable {
		b.failures++
		if b.state == BreakerHalfOpen || b.failures >= b.threshold {
			b.state = BreakerOpen
			b.openUntil = time.Now().Add(b.openFor)
		}
	}
	state := b.state
	b.mu.Unlock()
	b.setMetric(ctx, state)
}
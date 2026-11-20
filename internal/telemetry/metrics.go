package telemetry

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Global holds process metrics after Init.
var Global *Metrics

// Metrics implements design.md business and gateway SLIs.
type Metrics struct {
	ReservationRequests metric.Int64Counter
	ReaperLagSeconds    metric.Float64Histogram
	ResyncDrift         metric.Int64Counter
	PoolUtilization     metric.Float64Gauge
	BreakerState        metric.Int64Gauge

	poolUtilCb func(context.Context) []poolUtilSample
	mu         sync.RWMutex
}

// PoolUtilSample is one pool utilization observation.
type PoolUtilSample struct {
	Resource string
	Ratio    float64
}

type poolUtilSample = PoolUtilSample

func newMetrics(serviceName string) (*Metrics, error) {
	meter := otel.Meter("quota-challenge/" + serviceName)
	m := &Metrics{}

	var err error
	m.ReservationRequests, err = meter.Int64Counter("capacity.reservation.requests",
		metric.WithDescription("Reservation attempts by result"))
	if err != nil {
		return nil, err
	}
	m.ReaperLagSeconds, err = meter.Float64Histogram("capacity.reaper.lag",
		metric.WithDescription("Seconds between reservation expiry and reap"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	m.ResyncDrift, err = meter.Int64Counter("capacity.resync.drift",
		metric.WithDescription("Counter drift magnitude observed during resync"))
	if err != nil {
		return nil, err
	}
	m.PoolUtilization, err = meter.Float64Gauge("capacity.pool.utilization",
		metric.WithDescription("Pool utilization ratio allocated+reserved over total"))
	if err != nil {
		return nil, err
	}
	m.BreakerState, err = meter.Int64Gauge("gateway.region.breaker.state",
		metric.WithDescription("Per-region circuit breaker: 0=closed, 1=open, 2=half-open"))
	if err != nil {
		return nil, err
	}
	return m, nil
}

// RegisterPoolUtilization records pool ratios on each scrape.
func (m *Metrics) RegisterPoolUtilization(cb func(context.Context) []poolUtilSample) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.poolUtilCb = cb
	m.mu.Unlock()
}

// RecordPoolUtilization should be called periodically from the region service.
func (m *Metrics) RecordPoolUtilization(ctx context.Context) {
	if m == nil {
		return
	}
	m.mu.RLock()
	cb := m.poolUtilCb
	m.mu.RUnlock()
	if cb == nil {
		return
	}
	for _, s := range cb(ctx) {
		m.PoolUtilization.Record(ctx, s.Ratio, metric.WithAttributes(attribute.String("resource", s.Resource)))
	}
}

// RecordReservation increments capacity.reservation.requests.
func (m *Metrics) RecordReservation(ctx context.Context, result string) {
	if m == nil {
		return
	}
	m.ReservationRequests.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
}

// RecordReaperLag observes capacity.reaper.lag.
func (m *Metrics) RecordReaperLag(ctx context.Context, seconds float64) {
	if m == nil {
		return
	}
	m.ReaperLagSeconds.Record(ctx, seconds)
}

// RecordResyncDrift adds to capacity.resync.drift.
func (m *Metrics) RecordResyncDrift(ctx context.Context, drift int64) {
	if m == nil {
		return
	}
	if drift == 0 {
		return
	}
	m.ResyncDrift.Add(ctx, drift)
}

// SetBreakerState sets gateway.region.breaker.state for a region.
func (m *Metrics) SetBreakerState(ctx context.Context, region string, state int64) {
	if m == nil {
		return
	}
	m.BreakerState.Record(ctx, state, metric.WithAttributes(attribute.String("region", region)))
}

// PoolUtilFromTotals returns allocated+reserved over total.
func PoolUtilFromTotals(total, allocated, reserved int64) float64 {
	if total <= 0 {
		return 0
	}
	used := allocated + reserved
	if used < 0 {
		return 0
	}
	return float64(used) / float64(total)
}

// MakePoolUtilSample builds a pool utilization sample for callbacks.
func MakePoolUtilSample(resource string, ratio float64) PoolUtilSample {
	return PoolUtilSample{Resource: resource, Ratio: ratio}
}
package regionsvc

import (
	"context"

	"quota-challenge/internal/telemetry"
)

func (s *Service) Reap(ctx context.Context) (int64, error) {
	n, lag, err := s.repo.ReapExpired(ctx, 200)
	if err != nil {
		return n, err
	}
	if telemetry.Global != nil && n > 0 {
		telemetry.Global.RecordReaperLag(ctx, lag.Seconds())
	}
	return n, nil
}

func (s *Service) Resync(ctx context.Context, apply bool) (int64, error) {
	drift, err := s.repo.ResyncCounters(ctx, apply)
	if telemetry.Global != nil {
		telemetry.Global.RecordResyncDrift(ctx, drift)
	}
	return drift, err
}
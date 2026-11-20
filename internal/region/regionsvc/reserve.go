package regionsvc

import (
	"context"
	"errors"
	"sort"
	"time"

	"quota-challenge/internal/capacity"
	"quota-challenge/internal/telemetry"
)

func (s *Service) Reserve(ctx context.Context, req capacity.ReserveRequest) (*capacity.Service, bool, error) {
	hash := req.RequestHash
	if hash == "" {
		hash = HashReserve(req.Product, req.Items, req.TTLSeconds)
	}

	recordResult := func(result string) {
		if telemetry.Global != nil {
			telemetry.Global.RecordReservation(ctx, result)
		}
	}

	existing, err := s.repo.GetIdempotency(ctx, req.CustomerID, req.IdempotencyKey)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		if existing.RequestHash != hash {
			recordResult("mismatch")
			return nil, false, capacity.ErrIdempotencyMismatch
		}
		switch existing.Status {
		case string(capacity.IdempotencyCompleted):
			svc, _, err := s.repo.IdempotencyReplay(existing)
			recordResult("replay")
			return svc, true, err
		case string(capacity.IdempotencyRejected):
			status, body, err := s.repo.IdempotencyHTTPReplay(existing)
			if err != nil {
				recordResult("replay-error")
				return nil, true, capacity.ErrCapacityExceeded
			}
			recordResult("rejection-replay")
			return nil, true, &capacity.ReplayHTTPError{Status: status, Body: body}
		case "in_progress":
			if existing.LeaseExpires != nil && time.Now().UTC().Before(*existing.LeaseExpires) {
				recordResult("in-progress")
				return nil, false, capacity.ErrIdempotencyConflict
			}
		}
	}

	lease := s.cfg.IdempotencyLease
	if lease <= 0 {
		lease = DefaultConfig().IdempotencyLease
	}
	token, err := s.repo.ClaimIdempotency(ctx, req.CustomerID, req.IdempotencyKey, hash, lease)
	if err != nil {
		if errors.Is(err, capacity.ErrIdempotencyMismatch) {
			recordResult("mismatch")
			return nil, false, err
		}
		if errors.Is(err, capacity.ErrIdempotencyConflict) {
			existing, _ = s.repo.GetIdempotency(ctx, req.CustomerID, req.IdempotencyKey)
			if existing != nil {
				switch existing.Status {
				case string(capacity.IdempotencyCompleted):
					svc, _, err := s.repo.IdempotencyReplay(existing)
					recordResult("replay")
					return svc, true, err
				case string(capacity.IdempotencyRejected):
					status, body, replayErr := s.repo.IdempotencyHTTPReplay(existing)
					if replayErr != nil {
						recordResult("replay-error")
						return nil, true, capacity.ErrCapacityExceeded
					}
					recordResult("rejection-replay")
					return nil, true, &capacity.ReplayHTTPError{Status: status, Body: body}
				}
			}
			recordResult("in-progress")
			return nil, false, capacity.ErrIdempotencyConflict
		}
		return nil, false, err
	}

	productID, defaultTTL, maxTTL, err := s.repo.ProductTTLs(ctx, req.Product)
	if err != nil {
		recordResult("validation")
		return nil, false, capacity.ErrValidation
	}
	ttl := req.TTLSeconds
	if ttl == capacity.TTLDefaultUnset {
		if s.cfg.DefaultReservationTTL > 0 {
			ttl = s.cfg.DefaultReservationTTL
		} else {
			ttl = int64(defaultTTL)
		}
	}
	if ttl < 0 || (maxTTL > 0 && int(ttl) > maxTTL) {
		prob := capacity.ProblemDocument("validation-error", "Unprocessable Entity", 422, "ttl out of range")
		_ = s.repo.RecordRejectedService(ctx, req.CustomerID, productID, "ttl out of range")
		_ = s.repo.CompleteRejected(ctx, req.CustomerID, req.IdempotencyKey, token, prob)
		recordResult("validation")
		return nil, false, capacity.ErrValidation
	}

	var lines []capacity.ReserveLine
	for _, it := range req.Items {
		rt, err := s.repo.ResourceByName(ctx, it.Resource)
		if err != nil {
			recordResult("validation")
			return nil, false, capacity.ErrValidation
		}
		err = ValidateAmount(ResourceType{
			Name: rt.Name, MinAmount: rt.MinAmount, StepAmount: rt.StepAmount, MaxAmount: rt.MaxAmount, Discrete: rt.Discrete,
		}, it.Amount)
		if err != nil {
			_ = s.repo.RecordRejectedService(ctx, req.CustomerID, productID, err.Error())
			_ = s.repo.CompleteRejected(ctx, req.CustomerID, req.IdempotencyKey, token,
				capacity.ProblemDocument("validation-error", "Unprocessable Entity", 422, err.Error()))
			recordResult("validation")
			return nil, false, err
		}
		lines = append(lines, capacity.ReserveLine{ResourceTypeID: rt.ID, Name: rt.Name, Amount: it.Amount})
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].ResourceTypeID < lines[j].ResourceTypeID })

	tryCreate := func() (*capacity.Service, error) {
		return s.repo.ReserveCreate(ctx, req.CustomerID, productID, req.Product, lines, ttl, token, req.IdempotencyKey)
	}

	svc, err := tryCreate()
	if err != nil {
		if errors.Is(err, capacity.ErrQuotaExceeded) || errors.Is(err, capacity.ErrCapacityExceeded) {
			_ = s.repo.ReclaimExpiredInScope(ctx, rtIDs(lines))
			svc, err = tryCreate()
		}
	}
	if err != nil {
		if errors.Is(err, capacity.ErrQuotaExceeded) {
			_ = s.repo.RecordRejectedService(ctx, req.CustomerID, productID, "quota-exceeded")
			_ = s.repo.CompleteRejected(ctx, req.CustomerID, req.IdempotencyKey, token,
				capacity.ProblemDocument("quota-exceeded", "Conflict", 409, err.Error()))
			recordResult("quota-exceeded")
			return nil, false, err
		}
		if errors.Is(err, capacity.ErrCapacityExceeded) {
			_ = s.repo.RecordRejectedService(ctx, req.CustomerID, productID, "capacity-exceeded")
			_ = s.repo.CompleteRejected(ctx, req.CustomerID, req.IdempotencyKey, token,
				capacity.ProblemDocument("capacity-exceeded", "Conflict", 409, err.Error()))
			recordResult("capacity-exceeded")
			return nil, false, err
		}
		_ = s.repo.RecordRejectedService(ctx, req.CustomerID, productID, err.Error())
		_ = s.repo.CompleteRejected(ctx, req.CustomerID, req.IdempotencyKey, token,
			capacity.ProblemDocument("error", "Conflict", 409, err.Error()))
		recordResult("error")
		return nil, false, err
	}
	_ = s.repo.SaveCompletedReserve(ctx, req.CustomerID, req.IdempotencyKey, token, svc)
	recordResult("success")
	return svc, false, nil
}

func rtIDs(lines []capacity.ReserveLine) []int {
	ids := make([]int, len(lines))
	for i, l := range lines {
		ids[i] = l.ResourceTypeID
	}
	return ids
}
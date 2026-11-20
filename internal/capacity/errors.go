package capacity

import "errors"

var (
	ErrQuotaExceeded    = errors.New("quota exceeded")
	ErrCapacityExceeded = errors.New("capacity exceeded")
	ErrInvalidState     = errors.New("invalid state")
	ErrValidation       = errors.New("validation failed")
	ErrIdempotencyConflict = errors.New("idempotency in progress")
	ErrIdempotencyMismatch = errors.New("idempotency payload mismatch")
	ErrNotFound         = errors.New("not found")
	ErrReservationExpired = errors.New("reservation expired")
	ErrForbidden        = errors.New("forbidden")
)
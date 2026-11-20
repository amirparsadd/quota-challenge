package capacity

import "time"

// Confirmable reports whether the service may transition reserved → active.
func (s *Service) Confirmable(now time.Time) error {
	if s.State != StateReserved {
		return ErrInvalidState
	}
	if s.ExpiresAt != nil && !now.Before(*s.ExpiresAt) {
		return ErrReservationExpired
	}
	return nil
}

// Cancellable reports whether cancel is valid (reserved only).
func (s *Service) Cancellable() error {
	if s.State != StateReserved {
		return ErrInvalidState
	}
	return nil
}
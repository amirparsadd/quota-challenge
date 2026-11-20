package regionsvc

import (
	"context"

	"quota-challenge/internal/capacity"
)

func (s *Service) Confirm(ctx context.Context, customerID string, id string) (*capacity.Service, error) {
	sid, err := parseUUID(id)
	if err != nil {
		return nil, capacity.ErrValidation
	}
	return s.repo.ConfirmService(ctx, customerID, sid)
}
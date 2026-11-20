package regionsvc

import (
	"context"

	"quota-challenge/internal/capacity"
)

func (s *Service) Cancel(ctx context.Context, customerID, id string) (*capacity.Service, error) {
	sid, err := parseUUID(id)
	if err != nil {
		return nil, capacity.ErrValidation
	}
	return s.repo.CancelService(ctx, customerID, sid)
}
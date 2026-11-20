package regionsvc

import (
	"context"

	"quota-challenge/internal/capacity"
)

func (s *Service) Release(ctx context.Context, id string) (*capacity.Service, error) {
	sid, err := parseUUID(id)
	if err != nil {
		return nil, capacity.ErrValidation
	}
	return s.repo.ReleaseService(ctx, sid)
}
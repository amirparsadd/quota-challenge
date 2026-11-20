package regionsvc

import (
	"context"

	"quota-challenge/internal/capacity"
)

func (s *Service) SetPoolCapacity(ctx context.Context, resource string, total int64) (*capacity.Pool, error) {
	return s.repo.SetPoolTotal(ctx, resource, total)
}

func (s *Service) SetCustomerQuota(ctx context.Context, customerID, resource string, limit int64) error {
	return s.repo.SetCustomerQuota(ctx, customerID, resource, limit)
}

func (s *Service) GetPool(ctx context.Context, resource string) (*capacity.Pool, error) {
	return s.repo.GetPool(ctx, resource)
}
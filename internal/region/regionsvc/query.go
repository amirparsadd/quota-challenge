package regionsvc

import (
	"context"

	"quota-challenge/internal/capacity"
)

func (s *Service) Get(ctx context.Context, customerID, id string) (*capacity.Service, error) {
	sid, err := parseUUID(id)
	if err != nil {
		return nil, capacity.ErrValidation
	}
	return s.repo.GetService(ctx, customerID, sid)
}

func (s *Service) List(ctx context.Context, customerID string) ([]capacity.Service, error) {
	return s.repo.ListServices(ctx, customerID)
}

func (s *Service) Availability(ctx context.Context, customerID, product string) ([]capacity.AvailabilityEntry, error) {
	return s.repo.GetAvailability(ctx, customerID, product)
}

func (s *Service) Quota(ctx context.Context, customerID string) ([]capacity.QuotaEntry, error) {
	return s.repo.GetQuota(ctx, customerID)
}

func (s *Service) Products(ctx context.Context) ([]capacity.Product, error) {
	return s.repo.ListProducts(ctx)
}

func (s *Service) ListEvents(ctx context.Context, customerID, serviceID string) ([]capacity.ServiceEvent, error) {
	sid, err := parseUUID(serviceID)
	if err != nil {
		return nil, capacity.ErrValidation
	}
	return s.repo.ListServiceEvents(ctx, customerID, sid)
}
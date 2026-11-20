package grpcserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	regionquotav1 "quota-challenge/api/gen/regionquota/v1"
)

func (s *Server) SetPoolCapacity(ctx context.Context, req *regionquotav1.SetPoolCapacityRequest) (*regionquotav1.PoolResponse, error) {
	if err := denyGateway(ctx); err != nil {
		return nil, err
	}
	p, err := s.svc.SetPoolCapacity(ctx, req.Resource, req.Total)
	if err != nil {
		return nil, mapErr(err)
	}
	return &regionquotav1.PoolResponse{
		Resource: p.Resource, Total: p.Total, BaselineAllocated: p.BaselineAllocated, Allocated: p.Allocated, Reserved: p.Reserved,
	}, nil
}

func (s *Server) SetCustomerQuota(ctx context.Context, req *regionquotav1.SetCustomerQuotaRequest) (*regionquotav1.QuotaEntryResponse, error) {
	if err := denyGateway(ctx); err != nil {
		return nil, err
	}
	if err := s.svc.SetCustomerQuota(ctx, req.CustomerId, req.Resource, req.Limit); err != nil {
		return nil, mapErr(err)
	}
	entries, err := s.svc.Quota(ctx, req.CustomerId)
	if err != nil {
		return nil, mapErr(err)
	}
	for _, e := range entries {
		if e.Resource == req.Resource {
			ent := &regionquotav1.QuotaEntry{
				Resource: e.Resource, Unit: e.Unit, Allocated: e.Allocated,
				Reserved: e.Reserved, Provisioned: e.Provisioned,
			}
			if e.Provisioned {
				ent.Limit = &e.Limit
				rem := e.Limit - e.Allocated - e.Reserved
				ent.Remaining = &rem
			}
			return &regionquotav1.QuotaEntryResponse{Entry: ent}, nil
		}
	}
	return nil, status.Error(codes.NotFound, "resource")
}

func (s *Server) GetPool(ctx context.Context, req *regionquotav1.GetPoolRequest) (*regionquotav1.PoolResponse, error) {
	if err := denyGateway(ctx); err != nil {
		return nil, err
	}
	p, err := s.svc.GetPool(ctx, req.Resource)
	if err != nil {
		return nil, mapErr(err)
	}
	return &regionquotav1.PoolResponse{
		Resource: p.Resource, Total: p.Total, BaselineAllocated: p.BaselineAllocated, Allocated: p.Allocated, Reserved: p.Reserved,
	}, nil
}

func (s *Server) ResyncCounters(ctx context.Context, req *regionquotav1.ResyncCountersRequest) (*regionquotav1.ResyncCountersResponse, error) {
	if err := denyGateway(ctx); err != nil {
		return nil, err
	}
	drift, err := s.svc.Resync(ctx, req.Apply)
	if err != nil {
		return nil, mapErr(err)
	}
	return &regionquotav1.ResyncCountersResponse{Drift: drift, Applied: req.Apply}, nil
}

func (s *Server) ExpireServices(ctx context.Context, _ *regionquotav1.ExpireServicesRequest) (*regionquotav1.ExpireServicesResponse, error) {
	if err := denyGateway(ctx); err != nil {
		return nil, err
	}
	n, err := s.svc.Reap(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	return &regionquotav1.ExpireServicesResponse{ExpiredCount: n}, nil
}
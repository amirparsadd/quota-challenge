package grpcserver

import (
	"context"
	"errors"
	"time"

	"quota-challenge/internal/capacity"
	regionquotav1 "quota-challenge/api/gen/regionquota/v1"
)

func (s *Server) Reserve(ctx context.Context, req *regionquotav1.ReserveRequest) (*regionquotav1.ReserveResponse, error) {
	items := make([]capacity.ResourceLine, len(req.Items))
	for i, it := range req.Items {
		items[i] = capacity.ResourceLine{Resource: it.Resource, Amount: it.Amount}
	}
	ttl := capacity.TTLDefaultUnset
	if req.TtlSeconds != nil {
		ttl = req.GetTtlSeconds()
	}
	svcReq := capacity.ReserveRequest{
		CustomerID:     req.CustomerId,
		IdempotencyKey: req.IdempotencyKey,
		RequestHash:    req.RequestHash,
		Product:        req.Product,
		Items:          items,
		TTLSeconds:     ttl,
	}
	out, replay, err := s.svc.Reserve(ctx, svcReq)
	var replayHTTP *capacity.ReplayHTTPError
	if errors.As(err, &replayHTTP) {
		return &regionquotav1.ReserveResponse{
			Replayed:        true,
			RejectionReplay: true,
			HttpStatus:      int32(replayHTTP.Status),
			ProblemJson:     string(replayHTTP.Body),
		}, nil
	}
	if err != nil {
		return nil, mapErr(err)
	}
	return &regionquotav1.ReserveResponse{Service: toProtoService(out), Replayed: replay}, nil
}

func (s *Server) Confirm(ctx context.Context, req *regionquotav1.ConfirmRequest) (*regionquotav1.ServiceResponse, error) {
	out, err := s.svc.Confirm(ctx, req.CustomerId, req.ServiceId)
	if err != nil {
		return nil, mapErr(err)
	}
	return &regionquotav1.ServiceResponse{Service: toProtoService(out)}, nil
}

func (s *Server) Cancel(ctx context.Context, req *regionquotav1.CancelRequest) (*regionquotav1.ServiceResponse, error) {
	out, err := s.svc.Cancel(ctx, req.CustomerId, req.ServiceId)
	if err != nil {
		return nil, mapErr(err)
	}
	return &regionquotav1.ServiceResponse{Service: toProtoService(out)}, nil
}

func (s *Server) Release(ctx context.Context, req *regionquotav1.ReleaseRequest) (*regionquotav1.ServiceResponse, error) {
	out, err := s.svc.Release(ctx, req.ServiceId)
	if err != nil {
		return nil, mapErr(err)
	}
	return &regionquotav1.ServiceResponse{Service: toProtoService(out)}, nil
}

func (s *Server) GetService(ctx context.Context, req *regionquotav1.GetServiceRequest) (*regionquotav1.ServiceResponse, error) {
	out, err := s.svc.Get(ctx, req.CustomerId, req.ServiceId)
	if err != nil {
		return nil, mapErr(err)
	}
	return &regionquotav1.ServiceResponse{Service: toProtoService(out)}, nil
}

func (s *Server) ListServiceEvents(ctx context.Context, req *regionquotav1.ListServiceEventsRequest) (*regionquotav1.ListServiceEventsResponse, error) {
	list, err := s.svc.ListEvents(ctx, req.CustomerId, req.ServiceId)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]*regionquotav1.ServiceEvent, len(list))
	for i, ev := range list {
		out[i] = &regionquotav1.ServiceEvent{
			Id: ev.ID, EventType: ev.EventType, At: ev.At.UTC().Format(time.RFC3339),
		}
	}
	return &regionquotav1.ListServiceEventsResponse{Events: out}, nil
}

func (s *Server) ListServices(ctx context.Context, req *regionquotav1.ListServicesRequest) (*regionquotav1.ListServicesResponse, error) {
	list, err := s.svc.List(ctx, req.CustomerId)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]*regionquotav1.Service, len(list))
	for i := range list {
		out[i] = toProtoService(&list[i])
	}
	return &regionquotav1.ListServicesResponse{Services: out}, nil
}

func (s *Server) GetAvailability(ctx context.Context, req *regionquotav1.GetAvailabilityRequest) (*regionquotav1.AvailabilityResponse, error) {
	list, err := s.svc.Availability(ctx, req.CustomerId, req.GetProduct())
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]*regionquotav1.AvailabilityEntry, len(list))
	for i, e := range list {
		out[i] = &regionquotav1.AvailabilityEntry{
			Resource: e.Resource, Unit: e.Unit, PoolRemaining: e.PoolRemaining, QuotaRemaining: e.QuotaRemaining,
		}
	}
	return &regionquotav1.AvailabilityResponse{Entries: out}, nil
}

func (s *Server) GetQuota(ctx context.Context, req *regionquotav1.GetQuotaRequest) (*regionquotav1.QuotaResponse, error) {
	list, err := s.svc.Quota(ctx, req.CustomerId)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]*regionquotav1.QuotaEntry, len(list))
	for i, e := range list {
		ent := &regionquotav1.QuotaEntry{
			Resource: e.Resource, Unit: e.Unit, Allocated: e.Allocated,
			Reserved: e.Reserved, Provisioned: e.Provisioned,
		}
		if e.Provisioned {
			ent.Limit = &e.Limit
			rem := e.Limit - e.Allocated - e.Reserved
			ent.Remaining = &rem
		}
		out[i] = ent
	}
	return &regionquotav1.QuotaResponse{Entries: out}, nil
}

func (s *Server) ListProducts(ctx context.Context, _ *regionquotav1.ListProductsRequest) (*regionquotav1.ListProductsResponse, error) {
	list, err := s.svc.Products(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]*regionquotav1.Product, len(list))
	for i, p := range list {
		res := make([]*regionquotav1.ProductResource, len(p.Resources))
		for j, r := range p.Resources {
			res[j] = &regionquotav1.ProductResource{
				Resource: r.Resource, Unit: r.Unit, MinAmount: r.MinAmount, StepAmount: r.StepAmount,
			}
		}
		out[i] = &regionquotav1.Product{
			Name: p.Name, DefaultTtlSeconds: int32(p.DefaultTTLSeconds),
			MaxTtlSeconds: int32(p.MaxTTLSeconds), Resources: res,
		}
	}
	return &regionquotav1.ListProductsResponse{Products: out}, nil
}

func (s *Server) Health(ctx context.Context, _ *regionquotav1.HealthRequest) (*regionquotav1.HealthResponse, error) {
	return &regionquotav1.HealthResponse{Status: "ok", RegionId: s.regionID}, nil
}
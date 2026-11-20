package gateway

import (
	"context"
	"time"

	regionquotav1 "quota-challenge/api/gen/regionquota/v1"
	"quota-challenge/internal/regionclient"
)

// RegionBackend is one gateway region target with a circuit breaker.
type RegionBackend struct {
	Region  string
	Client  *regionclient.Client
	Breaker *Breaker
}

func newBackend(region string, client *regionclient.Client, breakerOpen time.Duration) *RegionBackend {
	return &RegionBackend{
		Region:  region,
		Client:  client,
		Breaker: NewBreaker(region, breakerOpen),
	}
}

func (b *RegionBackend) Reserve(ctx context.Context, req *regionquotav1.ReserveRequest) (*regionquotav1.ReserveResponse, error) {
	if err := b.Breaker.BeforeCall(ctx); err != nil {
		return nil, err
	}
	resp, err := b.Client.API.Reserve(ctx, req)
	b.Breaker.AfterCall(ctx, err)
	return resp, err
}

func (b *RegionBackend) call(ctx context.Context, fn func(context.Context) error) error {
	if err := b.Breaker.BeforeCall(ctx); err != nil {
		return err
	}
	err := fn(ctx)
	b.Breaker.AfterCall(ctx, err)
	return err
}

func (b *RegionBackend) Confirm(ctx context.Context, req *regionquotav1.ConfirmRequest) (*regionquotav1.ServiceResponse, error) {
	var resp *regionquotav1.ServiceResponse
	err := b.call(ctx, func(c context.Context) error {
		var e error
		resp, e = b.Client.API.Confirm(c, req)
		return e
	})
	return resp, err
}

func (b *RegionBackend) Cancel(ctx context.Context, req *regionquotav1.CancelRequest) (*regionquotav1.ServiceResponse, error) {
	var resp *regionquotav1.ServiceResponse
	err := b.call(ctx, func(c context.Context) error {
		var e error
		resp, e = b.Client.API.Cancel(c, req)
		return e
	})
	return resp, err
}

func (b *RegionBackend) GetService(ctx context.Context, req *regionquotav1.GetServiceRequest) (*regionquotav1.ServiceResponse, error) {
	var resp *regionquotav1.ServiceResponse
	err := b.call(ctx, func(c context.Context) error {
		var e error
		resp, e = b.Client.API.GetService(c, req)
		return e
	})
	return resp, err
}

func (b *RegionBackend) ListServices(ctx context.Context, req *regionquotav1.ListServicesRequest) (*regionquotav1.ListServicesResponse, error) {
	var resp *regionquotav1.ListServicesResponse
	err := b.call(ctx, func(c context.Context) error {
		var e error
		resp, e = b.Client.API.ListServices(c, req)
		return e
	})
	return resp, err
}

func (b *RegionBackend) ListServiceEvents(ctx context.Context, req *regionquotav1.ListServiceEventsRequest) (*regionquotav1.ListServiceEventsResponse, error) {
	var resp *regionquotav1.ListServiceEventsResponse
	err := b.call(ctx, func(c context.Context) error {
		var e error
		resp, e = b.Client.API.ListServiceEvents(c, req)
		return e
	})
	return resp, err
}

func (b *RegionBackend) GetAvailability(ctx context.Context, req *regionquotav1.GetAvailabilityRequest) (*regionquotav1.AvailabilityResponse, error) {
	var resp *regionquotav1.AvailabilityResponse
	err := b.call(ctx, func(c context.Context) error {
		var e error
		resp, e = b.Client.API.GetAvailability(c, req)
		return e
	})
	return resp, err
}

func (b *RegionBackend) GetQuota(ctx context.Context, req *regionquotav1.GetQuotaRequest) (*regionquotav1.QuotaResponse, error) {
	var resp *regionquotav1.QuotaResponse
	err := b.call(ctx, func(c context.Context) error {
		var e error
		resp, e = b.Client.API.GetQuota(c, req)
		return e
	})
	return resp, err
}

func (b *RegionBackend) ListProducts(ctx context.Context, req *regionquotav1.ListProductsRequest) (*regionquotav1.ListProductsResponse, error) {
	var resp *regionquotav1.ListProductsResponse
	err := b.call(ctx, func(c context.Context) error {
		var e error
		resp, e = b.Client.API.ListProducts(c, req)
		return e
	})
	return resp, err
}

func (b *RegionBackend) Health(ctx context.Context, req *regionquotav1.HealthRequest) (*regionquotav1.HealthResponse, error) {
	var resp *regionquotav1.HealthResponse
	err := b.call(ctx, func(c context.Context) error {
		var e error
		resp, e = b.Client.API.Health(c, req)
		return e
	})
	return resp, err
}
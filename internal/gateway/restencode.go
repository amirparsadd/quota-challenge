package gateway

import (
	regionquotav1 "quota-challenge/api/gen/regionquota/v1"
)

// REST JSON shapes match api/openapi/v1.yaml (encoding/json), not protojson.

type resourceItemREST struct {
	Resource string `json:"resource"`
	Amount   int64  `json:"amount"`
	Unit     string `json:"unit,omitempty"`
}

type serviceREST struct {
	ID            string             `json:"id"`
	CustomerID    string             `json:"customer_id"`
	Product       string             `json:"product"`
	State         string             `json:"state"`
	Items         []resourceItemREST `json:"items"`
	ExpiresAtUnix *int64             `json:"expires_at_unix,omitempty"`
	CreatedAt     string             `json:"created_at"`
	UpdatedAt     string             `json:"updated_at"`
}

type serviceResponseREST struct {
	Service serviceREST `json:"service"`
}

type listServicesResponseREST struct {
	Services []serviceREST `json:"services"`
}

type serviceEventREST struct {
	ID        int64  `json:"id"`
	EventType string `json:"event_type"`
	At        string `json:"at"`
}

type listServiceEventsResponseREST struct {
	Events []serviceEventREST `json:"events"`
}

type availabilityEntryREST struct {
	Resource       string `json:"resource"`
	Unit           string `json:"unit"`
	PoolRemaining  int64  `json:"pool_remaining"`
	QuotaRemaining int64  `json:"quota_remaining"`
}

type availabilityResponseREST struct {
	Entries []availabilityEntryREST `json:"entries"`
}

type quotaEntryREST struct {
	Resource    string `json:"resource"`
	Unit        string `json:"unit"`
	Limit       *int64 `json:"limit,omitempty"`
	Allocated   int64  `json:"allocated"`
	Reserved    int64  `json:"reserved"`
	Remaining   *int64 `json:"remaining,omitempty"`
	Provisioned bool   `json:"provisioned"`
}

type quotaResponseREST struct {
	Entries []quotaEntryREST `json:"entries"`
}

type productResourceREST struct {
	Resource   string `json:"resource"`
	Unit       string `json:"unit"`
	MinAmount  int64  `json:"min_amount"`
	StepAmount int64  `json:"step_amount"`
}

type productREST struct {
	Name              string                `json:"name"`
	DefaultTTLSeconds int32                 `json:"default_ttl_seconds"`
	MaxTTLSeconds     int32                 `json:"max_ttl_seconds"`
	Resources         []productResourceREST `json:"resources"`
}

type listProductsResponseREST struct {
	Products []productREST `json:"products"`
}

type fanoutAvailabilityResponseREST struct {
	Regions            []regionAvailabilityViewREST `json:"regions"`
	Partial            bool                         `json:"partial"`
	UnavailableRegions []string                     `json:"unavailable_regions"`
}

type regionAvailabilityViewREST struct {
	Region  string                  `json:"region"`
	Entries []availabilityEntryREST `json:"entries"`
}

type fanoutQuotaResponseREST struct {
	Regions            []regionQuotaViewREST `json:"regions"`
	Partial            bool                  `json:"partial"`
	UnavailableRegions []string              `json:"unavailable_regions"`
}

type regionQuotaViewREST struct {
	Region  string           `json:"region"`
	Entries []quotaEntryREST `json:"entries"`
}

type fanoutServicesResponseREST struct {
	Regions            []regionServicesViewREST `json:"regions"`
	Partial            bool                     `json:"partial"`
	UnavailableRegions []string                 `json:"unavailable_regions"`
}

type regionServicesViewREST struct {
	Region   string        `json:"region"`
	Services []serviceREST `json:"services"`
}

type fanoutProductsResponseREST struct {
	Regions            []regionProductsViewREST `json:"regions"`
	Partial            bool                     `json:"partial"`
	UnavailableRegions []string                 `json:"unavailable_regions"`
}

type regionProductsViewREST struct {
	Region   string        `json:"region"`
	Products []productREST `json:"products"`
}

func resourceItemFromProto(p *regionquotav1.ResourceLine) resourceItemREST {
	if p == nil {
		return resourceItemREST{}
	}
	return resourceItemREST{Resource: p.Resource, Amount: p.Amount, Unit: p.Unit}
}

func serviceFromProto(s *regionquotav1.Service) serviceREST {
	if s == nil {
		return serviceREST{Items: []resourceItemREST{}}
	}
	items := make([]resourceItemREST, len(s.Items))
	for i, it := range s.Items {
		items[i] = resourceItemFromProto(it)
	}
	out := serviceREST{
		ID: s.Id, CustomerID: s.CustomerId, Product: s.Product, State: s.State,
		Items: items, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
	if s.ExpiresAtUnix != nil {
		v := *s.ExpiresAtUnix
		out.ExpiresAtUnix = &v
	}
	return out
}

func encodeServiceResponse(resp *regionquotav1.ServiceResponse) serviceResponseREST {
	return serviceResponseREST{Service: serviceFromProto(resp.GetService())}
}

func encodeListServicesResponse(resp *regionquotav1.ListServicesResponse) listServicesResponseREST {
	svcs := resp.GetServices()
	out := make([]serviceREST, len(svcs))
	for i, s := range svcs {
		out[i] = serviceFromProto(s)
	}
	return listServicesResponseREST{Services: out}
}

func encodeListServiceEventsResponse(resp *regionquotav1.ListServiceEventsResponse) listServiceEventsResponseREST {
	events := resp.GetEvents()
	out := make([]serviceEventREST, len(events))
	for i, e := range events {
		out[i] = serviceEventREST{ID: e.Id, EventType: e.EventType, At: e.At}
	}
	return listServiceEventsResponseREST{Events: out}
}

func encodeAvailabilityResponse(resp *regionquotav1.AvailabilityResponse) availabilityResponseREST {
	entries := resp.GetEntries()
	out := make([]availabilityEntryREST, len(entries))
	for i, e := range entries {
		out[i] = availabilityEntryREST{
			Resource: e.Resource, Unit: e.Unit,
			PoolRemaining: e.PoolRemaining, QuotaRemaining: e.QuotaRemaining,
		}
	}
	return availabilityResponseREST{Entries: out}
}

func encodeQuotaResponse(resp *regionquotav1.QuotaResponse) quotaResponseREST {
	entries := resp.GetEntries()
	out := make([]quotaEntryREST, len(entries))
	for i, e := range entries {
		ent := quotaEntryREST{
			Resource: e.Resource, Unit: e.Unit, Allocated: e.Allocated,
			Reserved: e.Reserved, Provisioned: e.Provisioned,
		}
		if e.Limit != nil {
			v := *e.Limit
			ent.Limit = &v
		}
		if e.Remaining != nil {
			v := *e.Remaining
			ent.Remaining = &v
		}
		out[i] = ent
	}
	return quotaResponseREST{Entries: out}
}

func encodeListProductsResponse(resp *regionquotav1.ListProductsResponse) listProductsResponseREST {
	products := resp.GetProducts()
	out := make([]productREST, len(products))
	for i, p := range products {
		res := make([]productResourceREST, len(p.Resources))
		for j, r := range p.Resources {
			res[j] = productResourceREST{
				Resource: r.Resource, Unit: r.Unit, MinAmount: r.MinAmount, StepAmount: r.StepAmount,
			}
		}
		out[i] = productREST{
			Name: p.Name, DefaultTTLSeconds: p.DefaultTtlSeconds, MaxTTLSeconds: p.MaxTtlSeconds,
			Resources: res,
		}
	}
	return listProductsResponseREST{Products: out}
}

func encodeFanoutAvailability(resp *regionquotav1.FanoutAvailabilityResponse) fanoutAvailabilityResponseREST {
	regions := make([]regionAvailabilityViewREST, len(resp.Regions))
	for i, r := range resp.Regions {
		entries := make([]availabilityEntryREST, len(r.Entries))
		for j, e := range r.Entries {
			entries[j] = availabilityEntryREST{
				Resource: e.Resource, Unit: e.Unit,
				PoolRemaining: e.PoolRemaining, QuotaRemaining: e.QuotaRemaining,
			}
		}
		regions[i] = regionAvailabilityViewREST{Region: r.Region, Entries: entries}
	}
	unavail := resp.UnavailableRegions
	if unavail == nil {
		unavail = []string{}
	}
	return fanoutAvailabilityResponseREST{
		Regions: regions, Partial: resp.Partial, UnavailableRegions: unavail,
	}
}

func encodeFanoutQuota(resp *regionquotav1.FanoutQuotaResponse) fanoutQuotaResponseREST {
	regions := make([]regionQuotaViewREST, len(resp.Regions))
	for i, r := range resp.Regions {
		entries := make([]quotaEntryREST, len(r.Entries))
		for j, e := range r.Entries {
			ent := quotaEntryREST{
				Resource: e.Resource, Unit: e.Unit, Allocated: e.Allocated,
				Reserved: e.Reserved, Provisioned: e.Provisioned,
			}
			if e.Limit != nil {
				v := *e.Limit
				ent.Limit = &v
			}
			if e.Remaining != nil {
				v := *e.Remaining
				ent.Remaining = &v
			}
			entries[j] = ent
		}
		regions[i] = regionQuotaViewREST{Region: r.Region, Entries: entries}
	}
	unavail := resp.UnavailableRegions
	if unavail == nil {
		unavail = []string{}
	}
	return fanoutQuotaResponseREST{Regions: regions, Partial: resp.Partial, UnavailableRegions: unavail}
}

func encodeFanoutServices(resp *regionquotav1.FanoutServicesResponse) fanoutServicesResponseREST {
	regions := make([]regionServicesViewREST, len(resp.Regions))
	for i, r := range resp.Regions {
		services := make([]serviceREST, len(r.Services))
		for j, s := range r.Services {
			services[j] = serviceFromProto(s)
		}
		regions[i] = regionServicesViewREST{Region: r.Region, Services: services}
	}
	unavail := resp.UnavailableRegions
	if unavail == nil {
		unavail = []string{}
	}
	return fanoutServicesResponseREST{Regions: regions, Partial: resp.Partial, UnavailableRegions: unavail}
}

func encodeFanoutProducts(resp *regionquotav1.FanoutProductsResponse) fanoutProductsResponseREST {
	regions := make([]regionProductsViewREST, len(resp.Regions))
	for i, r := range resp.Regions {
		products := make([]productREST, len(r.Products))
		for j, p := range r.Products {
			res := make([]productResourceREST, len(p.Resources))
			for k, pr := range p.Resources {
				res[k] = productResourceREST{
					Resource: pr.Resource, Unit: pr.Unit, MinAmount: pr.MinAmount, StepAmount: pr.StepAmount,
				}
			}
			products[j] = productREST{
				Name: p.Name, DefaultTTLSeconds: p.DefaultTtlSeconds, MaxTTLSeconds: p.MaxTtlSeconds,
				Resources: res,
			}
		}
		regions[i] = regionProductsViewREST{Region: r.Region, Products: products}
	}
	unavail := resp.UnavailableRegions
	if unavail == nil {
		unavail = []string{}
	}
	return fanoutProductsResponseREST{Regions: regions, Partial: resp.Partial, UnavailableRegions: unavail}
}
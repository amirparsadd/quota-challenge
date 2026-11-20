package gateway

import (
	"net/http"
	"strings"
	"sync"

	regionquotav1 "quota-challenge/api/gen/regionquota/v1"
)

func (s *Server) fanoutAvailability(w http.ResponseWriter, r *http.Request) {
	cid, ok := customerID(r)
	if !ok {
		WriteProblem(w, http.StatusUnauthorized, "missing-customer", "X-Customer-ID required")
		return
	}
	product := strings.TrimSpace(r.URL.Query().Get("product"))
	out := &regionquotav1.FanoutAvailabilityResponse{
		Regions:            make([]*regionquotav1.RegionAvailabilityView, 0, len(s.regions)),
		UnavailableRegions: make([]string, 0),
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for id, b := range s.regions {
		wg.Add(1)
		go func(id string, b *RegionBackend) {
			defer wg.Done()
			ctx, cancel := s.fanoutContext(r.Context())
			defer cancel()
			req := &regionquotav1.GetAvailabilityRequest{CustomerId: cid}
			if product != "" {
				req.Product = &product
			}
			resp, err := b.GetAvailability(ctx, req)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				out.Partial = true
				out.UnavailableRegions = append(out.UnavailableRegions, id)
				return
			}
			entries := resp.Entries
			if entries == nil {
				entries = []*regionquotav1.AvailabilityEntry{}
			}
			out.Regions = append(out.Regions, &regionquotav1.RegionAvailabilityView{
				Region: id, Entries: entries,
			})
		}(id, b)
	}
	wg.Wait()
	writeJSON(w, http.StatusOK, encodeFanoutAvailability(out))
}

func (s *Server) fanoutQuota(w http.ResponseWriter, r *http.Request) {
	cid, ok := customerID(r)
	if !ok {
		WriteProblem(w, http.StatusUnauthorized, "missing-customer", "X-Customer-ID required")
		return
	}
	out := &regionquotav1.FanoutQuotaResponse{
		Regions:            make([]*regionquotav1.RegionQuotaView, 0, len(s.regions)),
		UnavailableRegions: make([]string, 0),
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for id, b := range s.regions {
		wg.Add(1)
		go func(id string, b *RegionBackend) {
			defer wg.Done()
			ctx, cancel := s.fanoutContext(r.Context())
			defer cancel()
			resp, err := b.GetQuota(ctx, &regionquotav1.GetQuotaRequest{CustomerId: cid})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				out.Partial = true
				out.UnavailableRegions = append(out.UnavailableRegions, id)
				return
			}
			entries := resp.Entries
			if entries == nil {
				entries = []*regionquotav1.QuotaEntry{}
			}
			out.Regions = append(out.Regions, &regionquotav1.RegionQuotaView{
				Region: id, Entries: entries,
			})
		}(id, b)
	}
	wg.Wait()
	writeJSON(w, http.StatusOK, encodeFanoutQuota(out))
}

func (s *Server) fanoutServices(w http.ResponseWriter, r *http.Request) {
	cid, ok := customerID(r)
	if !ok {
		WriteProblem(w, http.StatusUnauthorized, "missing-customer", "X-Customer-ID required")
		return
	}
	out := &regionquotav1.FanoutServicesResponse{
		Regions:            make([]*regionquotav1.RegionServicesView, 0, len(s.regions)),
		UnavailableRegions: make([]string, 0),
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for id, b := range s.regions {
		wg.Add(1)
		go func(id string, b *RegionBackend) {
			defer wg.Done()
			ctx, cancel := s.fanoutContext(r.Context())
			defer cancel()
			resp, err := b.ListServices(ctx, &regionquotav1.ListServicesRequest{CustomerId: cid})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				out.Partial = true
				out.UnavailableRegions = append(out.UnavailableRegions, id)
				return
			}
			services := resp.Services
			if services == nil {
				services = []*regionquotav1.Service{}
			}
			out.Regions = append(out.Regions, &regionquotav1.RegionServicesView{
				Region: id, Services: services,
			})
		}(id, b)
	}
	wg.Wait()
	writeJSON(w, http.StatusOK, encodeFanoutServices(out))
}
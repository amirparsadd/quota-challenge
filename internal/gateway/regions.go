package gateway

import (
	"net/http"

	regionquotav1 "quota-challenge/api/gen/regionquota/v1"
)

func (s *Server) handleRegions(w http.ResponseWriter, r *http.Request) {
	type item struct {
		Region string `json:"region"`
		Status string `json:"status"`
	}
	out := make([]item, 0, len(s.regions))
	for id, b := range s.regions {
		ctx, cancel := s.healthContext(r.Context())
		_, err := b.Health(ctx, &regionquotav1.HealthRequest{})
		cancel()
		st := "available"
		switch {
		case err != nil:
			st = "unavailable"
		case b.Breaker.State() == BreakerOpen:
			st = "degraded"
		}
		out = append(out, item{Region: id, Status: st})
	}
	writeJSON(w, http.StatusOK, map[string]any{"regions": out})
}
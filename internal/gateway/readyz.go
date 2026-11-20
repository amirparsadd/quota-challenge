package gateway

import (
	"net/http"

	regionquotav1 "quota-challenge/api/gen/regionquota/v1"
)

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if len(s.regions) == 0 {
		WriteProblem(w, http.StatusServiceUnavailable, "not-ready", "no regions configured")
		return
	}
	for id, b := range s.regions {
		ctx, cancel := s.healthContext(r.Context())
		_, err := b.Health(ctx, &regionquotav1.HealthRequest{})
		cancel()
		if err != nil {
			WriteProblem(w, http.StatusServiceUnavailable, "region-unavailable", id+": "+err.Error())
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
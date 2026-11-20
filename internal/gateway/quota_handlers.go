package gateway

import (
	"net/http"
	"strings"

	regionquotav1 "quota-challenge/api/gen/regionquota/v1"
)

func (s *Server) regionAvailability(w http.ResponseWriter, r *http.Request) {
	region := r.PathValue("region")
	cid, ok := customerID(r)
	if !ok {
		WriteProblem(w, http.StatusUnauthorized, "missing-customer", "X-Customer-ID required")
		return
	}
	b, ok := s.backend(region)
	if !ok {
		WriteProblem(w, http.StatusNotFound, "unknown-region", region)
		return
	}
	product := strings.TrimSpace(r.URL.Query().Get("product"))
	req := &regionquotav1.GetAvailabilityRequest{CustomerId: cid}
	if product != "" {
		req.Product = &product
	}
	ctx, cancel := s.rpcContext(r.Context())
	defer cancel()
	resp, err := b.GetAvailability(ctx, req)
	if err != nil {
		WriteGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, encodeAvailabilityResponse(resp))
}

func (s *Server) regionQuota(w http.ResponseWriter, r *http.Request) {
	region := r.PathValue("region")
	cid, ok := customerID(r)
	if !ok {
		WriteProblem(w, http.StatusUnauthorized, "missing-customer", "X-Customer-ID required")
		return
	}
	b, ok := s.backend(region)
	if !ok {
		WriteProblem(w, http.StatusNotFound, "unknown-region", region)
		return
	}
	ctx, cancel := s.rpcContext(r.Context())
	defer cancel()
	resp, err := b.GetQuota(ctx, &regionquotav1.GetQuotaRequest{CustomerId: cid})
	if err != nil {
		WriteGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, encodeQuotaResponse(resp))
}
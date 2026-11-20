package gateway

import (
	"net/http"

	regionquotav1 "quota-challenge/api/gen/regionquota/v1"
)

func (s *Server) regionProducts(w http.ResponseWriter, r *http.Request) {
	region := r.PathValue("region")
	if _, ok := customerID(r); !ok {
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
	resp, err := b.ListProducts(ctx, &regionquotav1.ListProductsRequest{})
	if err != nil {
		WriteGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, encodeListProductsResponse(resp))
}

func (s *Server) fanoutProducts(w http.ResponseWriter, r *http.Request) {
	if _, ok := customerID(r); !ok {
		WriteProblem(w, http.StatusUnauthorized, "missing-customer", "X-Customer-ID required")
		return
	}
	out := &regionquotav1.FanoutProductsResponse{
		Regions:            make([]*regionquotav1.RegionProductsView, 0, len(s.regions)),
		UnavailableRegions: make([]string, 0),
	}
	for id, b := range s.regions {
		resp, err := b.ListProducts(r.Context(), &regionquotav1.ListProductsRequest{})
		if err != nil {
			out.Partial = true
			out.UnavailableRegions = append(out.UnavailableRegions, id)
			continue
		}
		products := resp.Products
		if products == nil {
			products = []*regionquotav1.Product{}
		}
		out.Regions = append(out.Regions, &regionquotav1.RegionProductsView{
			Region: id, Products: products,
		})
	}
	writeJSON(w, http.StatusOK, encodeFanoutProducts(out))
}
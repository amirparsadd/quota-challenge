package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"quota-challenge/internal/obslog"
	"quota-challenge/internal/regionclient"
)

type Server struct {
	regions     map[string]*RegionBackend
	mux         *http.ServeMux
	createRates *RateLimiter
	timeouts    Timeouts
}

func NewServer(backends map[string]*regionclient.Client, timeouts Timeouts) *Server {
	if timeouts == (Timeouts{}) {
		timeouts = DefaultTimeouts()
	}
	regions := make(map[string]*RegionBackend, len(backends))
	for id, c := range backends {
		regions[id] = newBackend(id, c, timeouts.BreakerOpen)
	}
	s := &Server{regions: regions, mux: http.NewServeMux(), timeouts: timeouts}
	s.routes()
	return s
}

func (s *Server) rpcContext(parent context.Context) (context.Context, context.CancelFunc) {
	return rpcContext(parent, s.timeouts.RegionRPC)
}

func (s *Server) fanoutContext(parent context.Context) (context.Context, context.CancelFunc) {
	return rpcContext(parent, s.timeouts.FanoutRPC)
}

func (s *Server) healthContext(parent context.Context) (context.Context, context.CancelFunc) {
	return rpcContext(parent, s.timeouts.HealthRPC)
}

func (s *Server) Handler() http.Handler {
	return otelhttp.NewHandler(obslog.HTTPMiddleware(s.mux), "quota-gateway")
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /openapi/v1.yaml", s.handleOpenAPI)
	s.mux.HandleFunc("GET /docs", s.handleScalarDocs)
	s.mux.HandleFunc("GET /docs/", s.handleScalarDocs)
	s.mux.HandleFunc("GET /v1/problems", s.handleListProblems)
	s.mux.HandleFunc("GET /v1/problems/{slug}", s.handleGetProblem)
	s.mux.HandleFunc("GET /v1/regions", s.handleRegions)
	s.mux.HandleFunc("GET /v1/products", s.fanoutProducts)
	s.mux.HandleFunc("GET /v1/regions/{region}/products", s.regionProducts)
	s.mux.HandleFunc("GET /v1/availability", s.fanoutAvailability)
	s.mux.HandleFunc("GET /v1/quotas", s.fanoutQuota)
	s.mux.HandleFunc("GET /v1/services", s.fanoutServices)
	s.mux.HandleFunc("GET /v1/regions/{region}/availability", s.regionAvailability)
	s.mux.HandleFunc("GET /v1/regions/{region}/quota", s.regionQuota)
	s.mux.HandleFunc("GET /v1/regions/{region}/services", s.listServices)
	s.mux.HandleFunc("POST /v1/regions/{region}/services", s.withCreateRateLimit(s.createService))
	s.mux.HandleFunc("GET /v1/regions/{region}/services/{id}", s.getService)
	s.mux.HandleFunc("GET /v1/regions/{region}/services/{id}/events", s.listServiceEvents)
	s.mux.HandleFunc("PATCH /v1/regions/{region}/services/{id}", s.confirmService)
	s.mux.HandleFunc("DELETE /v1/regions/{region}/services/{id}", s.cancelService)
}

func customerID(r *http.Request) (string, bool) {
	id := strings.TrimSpace(r.Header.Get("X-Customer-ID"))
	return id, id != ""
}

func (s *Server) backend(region string) (*RegionBackend, bool) {
	b, ok := s.regions[region]
	return b, ok
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

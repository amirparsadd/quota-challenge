package gateway

import "net/http"

// OpsHandler serves liveness, readiness, and Prometheus metrics on the internal ops port.
func (s *Server) OpsHandler(metrics http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /readyz", s.handleReadyz)
	if metrics != nil {
		mux.Handle("GET /metrics", metrics)
	} else {
		mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		})
	}
	return mux
}
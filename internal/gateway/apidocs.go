package gateway

import (
	_ "embed"
	"net/http"

	"quota-challenge/api/openapi"
)

//go:embed scalar.html
var scalarHTML []byte

func (s *Server) handleOpenAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(openapi.V1)
}

func (s *Server) handleScalarDocs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(scalarHTML)
}
package gateway

import (
	"net/http"
	"strings"
)

type problemTypeDoc struct {
	Slug        string `json:"slug"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Status      int    `json:"status"`
	Description string `json:"description"`
}

var problemCatalog = []problemTypeDoc{
	{Slug: "validation-error", Title: "Unprocessable Entity", Status: 422, Description: "Request body or headers failed validation (min/step/max, TTL, product)."},
	{Slug: "quota-exceeded", Title: "Conflict", Status: 409, Description: "Customer quota would be exceeded by the reservation."},
	{Slug: "capacity-exceeded", Title: "Conflict", Status: 409, Description: "Regional pool capacity would be exceeded."},
	{Slug: "invalid-state", Title: "Conflict", Status: 409, Description: "Lifecycle transition is not allowed for the service state."},
	{Slug: "reservation-expired", Title: "Conflict", Status: 409, Description: "Reserved service expired before confirm."},
	{Slug: "idempotency-in-progress", Title: "Conflict", Status: 409, Description: "Another request with the same Idempotency-Key is in progress; retry after Retry-After."},
	{Slug: "idempotency-mismatch", Title: "Unprocessable Entity", Status: 422, Description: "Idempotency-Key reused with a different request payload."},
	{Slug: "not-found", Title: "Not Found", Status: 404, Description: "Region, service, or resource was not found."},
	{Slug: "unknown-region", Title: "Not Found", Status: 404, Description: "Region is not configured on this gateway."},
	{Slug: "region-unavailable", Title: "Service Unavailable", Status: 503, Description: "Region backend is unavailable or breaker is open."},
	{Slug: "missing-customer", Title: "Unauthorized", Status: 401, Description: "X-Customer-ID header is required."},
	{Slug: "missing-headers", Title: "Bad Request", Status: 400, Description: "Required headers (e.g. Idempotency-Key on create) are missing."},
	{Slug: "outcome-unknown", Title: "Gateway Timeout", Status: 504, Description: "Upstream timed out; retry with the same Idempotency-Key."},
	{Slug: "forbidden", Title: "Forbidden", Status: 403, Description: "Caller is not allowed to perform this action."},
	{Slug: "internal-error", Title: "Internal Server Error", Status: 500, Description: "Unexpected server error."},
}

func init() {
	for i := range problemCatalog {
		problemCatalog[i].Type = capacityProblemType(problemCatalog[i].Slug)
	}
}

func capacityProblemType(slug string) string {
	return "/v1/problems/" + slug
}

func (s *Server) handleListProblems(w http.ResponseWriter, _ *http.Request) {
	out := make([]problemTypeDoc, len(problemCatalog))
	copy(out, problemCatalog)
	writeJSON(w, http.StatusOK, map[string]any{"problems": out})
}

func (s *Server) handleGetProblem(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimSpace(r.PathValue("slug"))
	for _, p := range problemCatalog {
		if p.Slug == slug {
			writeJSON(w, http.StatusOK, p)
			return
		}
	}
	WriteProblem(w, http.StatusNotFound, "not-found", "unknown problem type")
}
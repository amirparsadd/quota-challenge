package gateway

import (
	"encoding/json"
	"net/http"
	"strconv"

	"quota-challenge/internal/capacity"
)

type Problem struct {
	Type   string         `json:"type"`
	Title  string         `json:"title"`
	Status int            `json:"status"`
	Detail string         `json:"detail"`
	Region string         `json:"region,omitempty"`
	Extra  map[string]any `json:"-"`
}

type problemEnvelope struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
	Region string `json:"region,omitempty"`
}

func WriteProblem(w http.ResponseWriter, status int, slug, detail string) {
	WriteProblemExt(w, status, slug, detail, "")
}

func WriteProblemExt(w http.ResponseWriter, status int, slug, detail, region string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problemEnvelope{
		Type:   capacity.ProblemTypeURI(slug),
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
		Region: region,
	})
}

// WriteProblemWithRetry sets Retry-After for idempotency conflicts (design).
func WriteProblemWithRetry(w http.ResponseWriter, status int, slug, detail string, retryAfterSec int) {
	if retryAfterSec > 0 {
		w.Header().Set("Retry-After", jsonRetryAfter(retryAfterSec))
	}
	WriteProblem(w, status, slug, detail)
}

func jsonRetryAfter(sec int) string {
	return strconv.Itoa(sec)
}

// WriteRawProblem writes a stored problem JSON body (idempotent replay).
func WriteRawProblem(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
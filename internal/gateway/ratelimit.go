package gateway

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

const (
	rateLimitRPS   = 50
	rateLimitBurst = 100
)

// RateLimiter caps hot creates per customer (design: rejection storm / sold-out retries).
type RateLimiter struct {
	mu       sync.Mutex
	perCust  map[string]*rate.Limiter
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{perCust: make(map[string]*rate.Limiter)}
}

func (rl *RateLimiter) allow(customerID string) bool {
	rl.mu.Lock()
	lim, ok := rl.perCust[customerID]
	if !ok {
		lim = rate.NewLimiter(rate.Limit(rateLimitRPS), rateLimitBurst)
		rl.perCust[customerID] = lim
	}
	rl.mu.Unlock()
	return lim.Allow()
}

func (s *Server) withCreateRateLimit(next http.HandlerFunc) http.HandlerFunc {
	if s.createRates == nil {
		s.createRates = NewRateLimiter()
	}
	return func(w http.ResponseWriter, r *http.Request) {
		cid, ok := customerID(r)
		if ok && !s.createRates.allow(cid) {
			WriteProblemWithRetry(w, http.StatusTooManyRequests, "rate-limited", "too many create requests", 1)
			return
		}
		next(w, r)
	}
}
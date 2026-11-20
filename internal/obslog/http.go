package obslog

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

// HTTPMiddleware logs each request at debug; 4xx at warn; 5xx at error (business 4xx are warn, not error).
func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		ms := time.Since(start).Milliseconds()
		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", ms,
			"bytes", sw.bytes,
		}
		if cid := strings.TrimSpace(r.Header.Get("X-Customer-ID")); cid != "" {
			attrs = append(attrs, "customer_id", cid)
		}
		if region := r.PathValue("region"); region != "" {
			attrs = append(attrs, "region", region)
		}
		switch {
		case sw.status >= 500:
			slog.Error("http request", attrs...)
		case sw.status >= 400:
			slog.Warn("http request", attrs...)
		default:
			slog.Debug("http request", attrs...)
		}
	})
}
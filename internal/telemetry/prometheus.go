package telemetry

import (
	"net/http"

	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	prom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

var promRegisterer promclient.Gatherer

func newPrometheusReader() (metric.Reader, error) {
	reg := promclient.NewRegistry()
	exp, err := prom.New(prom.WithRegisterer(reg))
	if err != nil {
		return nil, err
	}
	promRegisterer = reg
	return exp, nil
}

// PrometheusHandler returns the ops /metrics handler after Init (nil if telemetry disabled).
func PrometheusHandler() http.Handler {
	if promRegisterer == nil {
		return nil
	}
	return promhttp.HandlerFor(promRegisterer, promhttp.HandlerOpts{})
}
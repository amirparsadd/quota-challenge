package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Shutdown flushes telemetry providers.
type Shutdown func(context.Context) error

// Init configures global OTel trace and metric providers for the process.
// Set OTEL_SDK_DISABLED=true to no-op. OTLP is used when OTEL_EXPORTER_OTLP_ENDPOINT
// is set; otherwise metrics and traces go to stdout for local compose.
func Init(ctx context.Context, serviceName string) (Shutdown, error) {
	if strings.EqualFold(os.Getenv("OTEL_SDK_DISABLED"), "true") {
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	var shutdowns []Shutdown

	tp, tpShutdown, err := newTracerProvider(ctx, res)
	if err != nil {
		return nil, err
	}
	otel.SetTracerProvider(tp)
	shutdowns = append(shutdowns, tpShutdown)

	mp, mpShutdown, err := newMeterProvider(ctx, res)
	if err != nil {
		_ = tpShutdown(ctx)
		return nil, err
	}
	otel.SetMeterProvider(mp)
	shutdowns = append(shutdowns, mpShutdown)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	m, err := newMetrics(serviceName)
	if err != nil {
		return nil, err
	}
	Global = m

	slog.Info("telemetry ready", "service", serviceName, "otlp", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "")

	return func(ctx context.Context) error {
		var errs []error
		for _, fn := range shutdowns {
			if fn != nil {
				errs = append(errs, fn(ctx))
			}
		}
		return errors.Join(errs...)
	}, nil
}

func newTracerProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, Shutdown, error) {
	if ep := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")); ep != "" {
		exp, err := otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(trimHost(ep)),
			otlptracehttp.WithInsecure(),
		)
		if err != nil {
			return nil, nil, err
		}
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithBatcher(exp),
		)
		return tp, tp.Shutdown, nil
	}
	exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exp),
	)
	return tp, tp.Shutdown, nil
}

func newMeterProvider(ctx context.Context, res *resource.Resource) (*metric.MeterProvider, Shutdown, error) {
	opts := []metric.Option{metric.WithResource(res)}

	promReader, err := newPrometheusReader()
	if err != nil {
		return nil, nil, err
	}
	opts = append(opts, metric.WithReader(promReader))

	if ep := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")); ep != "" {
		exp, err := otlpmetrichttp.New(ctx,
			otlpmetrichttp.WithEndpoint(trimHost(ep)),
			otlpmetrichttp.WithInsecure(),
		)
		if err != nil {
			return nil, nil, err
		}
		opts = append(opts, metric.WithReader(metric.NewPeriodicReader(exp, metric.WithInterval(15*time.Second))))
	} else {
		exp, err := stdoutmetric.New()
		if err != nil {
			return nil, nil, err
		}
		opts = append(opts, metric.WithReader(metric.NewPeriodicReader(exp, metric.WithInterval(15*time.Second))))
	}
	mp := metric.NewMeterProvider(opts...)
	return mp, mp.Shutdown, nil
}

func trimHost(ep string) string {
	ep = strings.TrimPrefix(ep, "https://")
	ep = strings.TrimPrefix(ep, "http://")
	return ep
}
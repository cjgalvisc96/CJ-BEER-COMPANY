// Package telemetry wires OpenTelemetry: distributed traces (OTLP/HTTP
// export when an endpoint is configured) and Prometheus metrics served at
// /metrics. Both degrade to no-ops when unconfigured — observability is
// opt-in like every other production concern here.
package telemetry

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// InitTracing installs the global tracer provider and W3C propagation.
// With an empty endpoint only propagation is set (spans are no-ops); the
// returned shutdown is safe to call either way.
func InitTracing(ctx context.Context, endpoint, serviceName string) func(context.Context) error {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	if endpoint == "" {
		return func(context.Context) error { return nil }
	}
	// The HTTP exporter's construction cannot fail (a malformed endpoint
	// is logged and falls back to the default; its client Start is a
	// no-op), so the error is unreachable.
	exporter, _ := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint))
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL, semconv.ServiceName(serviceName),
		)),
	)
	otel.SetTracerProvider(provider)
	return provider.Shutdown
}

// InitMetrics installs a Prometheus-backed global meter provider and
// returns the handler to mount at /metrics.
func InitMetrics(serviceName string) (http.Handler, error) {
	registry := prometheus.NewRegistry()
	return initMetrics(serviceName, registry, registry)
}

func initMetrics(serviceName string, registerer prometheus.Registerer, gatherer prometheus.Gatherer) (http.Handler, error) {
	exporter, err := otelprom.New(otelprom.WithRegisterer(registerer))
	if err != nil {
		return nil, err
	}
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
		sdkmetric.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL, semconv.ServiceName(serviceName),
		)),
	)
	otel.SetMeterProvider(provider)
	return promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}), nil
}

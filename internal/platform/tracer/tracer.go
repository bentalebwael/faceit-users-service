package tracer

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/bentalebwael/faceit-users-service/internal/config"
)

const (
	tracerName = "faceit-users-service"
)

// NewTracerProvider creates and configures a new OpenTelemetry TracerProvider.
func NewTracerProvider(cfg *config.Config) (*sdktrace.TracerProvider, error) {
	// If no endpoint is provided, return a no-op tracer provider
	if cfg.Trace.ExporterEndpoint == "" {
		return sdktrace.NewTracerProvider(), nil
	}

	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.Trace.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Remove any protocol prefix from the endpoint
	endpoint := cfg.Trace.ExporterEndpoint
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	// Set global propagator to tracecontext (standard W3C Trace Context)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tracerProvider, nil
}

func Shutdown(ctx context.Context, tp *sdktrace.TracerProvider) error {
	if tp == nil {
		return nil
	}

	// Use the provided context directly to respect its timeout
	err := tp.Shutdown(ctx)
	if ctx.Err() != nil {
		// Return context error to indicate timeout
		return ctx.Err()
	}
	return err
}

// GetTracer returns a named tracer from the global provider
func GetTracer() trace.Tracer {
	return otel.GetTracerProvider().Tracer(tracerName)
}

// StartSpan starts a new span with the given name and options
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return GetTracer().Start(ctx, name, opts...)
}

// WithAttributes adds attributes to a span
func WithAttributes(span trace.Span, attributes ...attribute.KeyValue) {
	span.SetAttributes(attributes...)
}

// AddError adds an error to a span
func AddError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SpanFromContext returns the current span from the context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

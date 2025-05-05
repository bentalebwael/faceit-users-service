package tracer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bentalebwael/faceit-users-service/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func createTracerConfig(endpoint string, serviceName string) *config.Config {
	return &config.Config{
		Trace: config.TraceConfig{
			ExporterEndpoint: endpoint,
			ServiceName:      serviceName,
		},
	}
}

func TestNewTracerProvider_NoEndpoint(t *testing.T) {
	cfg := createTracerConfig("", "test-service-no-endpoint")

	tp, err := NewTracerProvider(cfg)

	require.NoError(t, err, "NewTracerProvider should not error when endpoint is empty")
	require.NotNil(t, tp, "NewTracerProvider should return a non-nil provider even if endpoint is empty")

	_ = tp.Shutdown(context.Background())
}

func TestShutdown_NilProvider(t *testing.T) {
	ctx := context.Background()
	err := Shutdown(ctx, nil)

	assert.NoError(t, err, "Shutdown should not return an error for a nil provider")
}

func TestShutdown_Success(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	require.NotNil(t, tp)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := Shutdown(ctx, tp)

	assert.NoError(t, err, "Shutdown should successfully shut down a default provider")
}

func TestGetTracer(t *testing.T) {
	tracer := GetTracer()
	assert.NotNil(t, tracer, "GetTracer should return a non-nil tracer")
}

func TestWithAttributes(t *testing.T) {
	ctx := context.Background()
	span := trace.SpanFromContext(ctx)

	// Test adding attributes
	WithAttributes(span, attribute.String("key", "value"))
	// Note: We can't verify the attributes directly as they're internal to the span,
	// but we can verify the function doesn't panic
}

func TestAddError(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "error-span")
	defer span.End()

	testErr := fmt.Errorf("test error")
	AddError(span, testErr)
	// Note: We can't verify the error status directly as it's internal to the span,
	// but we can verify the function doesn't panic with a real error

	AddError(span, nil)
	// Verify nil error doesn't panic
}

func TestSpanFromContext(t *testing.T) {
	ctx := context.Background()
	span := SpanFromContext(ctx)
	assert.NotNil(t, span, "SpanFromContext should never return nil")
}

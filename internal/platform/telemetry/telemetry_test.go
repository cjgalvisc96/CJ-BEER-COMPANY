package telemetry_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/platform/telemetry"
)

func TestInitTracingDisabledIsANoOp(t *testing.T) {
	shutdown := telemetry.InitTracing(context.Background(), "", "svc")

	assert.NoError(t, shutdown(context.Background()))
}

func TestInitTracingWithEndpoint(t *testing.T) {
	// The exporter connects lazily; construction and shutdown must work
	// without a collector listening.
	shutdown := telemetry.InitTracing(context.Background(), "http://127.0.0.1:1", "svc")

	_ = shutdown(context.Background()) // flush may fail: nothing listens — fine
}

func TestInitMetricsServesRegistry(t *testing.T) {
	handler, err := telemetry.InitMetrics("svc")
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/metrics", nil))

	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "target_info",
		"the OTel resource is exported")
}

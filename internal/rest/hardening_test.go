package rest_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/telemetry"
	"github.com/cjgalvisc96/cj-beer-company/internal/rest"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales"
	salesservices "github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/services"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses"
	warehousesservices "github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/services"
)

func routerWith(t *testing.T, opts rest.Options) http.Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	bus := muflone.NewServiceBus(slog.Default())
	t.Cleanup(func() { _ = bus.Close() })
	return rest.NewRouter(
		slog.Default(),
		sales.NewFacade(bus, salesservices.NewSalesOrderService()),
		warehouses.NewFacade(bus, warehousesservices.NewAvailabilityService()),
		opts,
	)
}

func TestListEndpointsArePaginated(t *testing.T) {
	router := routerWith(t, rest.Options{})

	recorder := do(t, router, http.MethodGet, "/v1/sales?limit=5&offset=10", "")
	require.Equal(t, http.StatusOK, recorder.Code)
	var envelope struct {
		Items  []any `json:"items"`
		Limit  int   `json:"limit"`
		Offset int   `json:"offset"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	assert.Equal(t, 5, envelope.Limit)
	assert.Equal(t, 10, envelope.Offset)
	assert.Empty(t, envelope.Items)

	// Out-of-range values clamp instead of erroring.
	recorder = do(t, router, http.MethodGet, "/v1/warehouses/availability?limit=9999&offset=-3", "")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	assert.Equal(t, 200, envelope.Limit, "clamped to the max")
	assert.Equal(t, 0, envelope.Offset)
}

func TestRateLimitReturns429(t *testing.T) {
	router := routerWith(t, rest.Options{RateLimitRPS: 1, RateLimitBurst: 2})

	assert.Equal(t, http.StatusOK, do(t, router, http.MethodGet, "/healthz", "").Code)
	assert.Equal(t, http.StatusOK, do(t, router, http.MethodGet, "/healthz", "").Code)
	assert.Equal(t, http.StatusTooManyRequests,
		do(t, router, http.MethodGet, "/healthz", "").Code, "burst exhausted")
}

func TestBodyLimitReturns413(t *testing.T) {
	router := routerWith(t, rest.Options{MaxBodyBytes: 64})

	big := `{"sales_order_number":"1","customer_name":"` +
		string(make([]byte, 200)) + `","rows":[]}`
	assert.Equal(t, http.StatusRequestEntityTooLarge,
		do(t, router, http.MethodPost, "/v1/sales", big).Code)
	assert.Equal(t, http.StatusOK,
		do(t, router, http.MethodGet, "/healthz", "").Code, "small requests pass")
}

func TestMetricsEndpointServesPrometheus(t *testing.T) {
	handler, err := telemetry.InitMetrics("cj-beer-company-test")
	require.NoError(t, err)
	router := routerWith(t, rest.Options{MetricsHandler: handler, TracingEnabled: true})

	// Generate one measured request, then scrape.
	require.Equal(t, http.StatusOK, do(t, router, http.MethodGet, "/healthz", "").Code)
	recorder := do(t, router, http.MethodGet, "/metrics", "")

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "http_server_requests")
}

// TestRateLimiterCannotBeSpoofedByDefault: with no trusted proxies, the
// client IP comes from the connection — X-Forwarded-For is ignored, so an
// attacker cannot mint fresh rate-limit buckets per request.
func TestRateLimiterCannotBeSpoofedByDefault(t *testing.T) {
	router := routerWith(t, rest.Options{RateLimitRPS: 1, RateLimitBurst: 2})

	for i, want := range []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests} {
		recorder := doWithHeaders(t, router, http.MethodGet, "/healthz", "",
			map[string]string{"X-Forwarded-For": "10.9.9." + strconv.Itoa(i)})
		assert.Equal(t, want, recorder.Code, "request %d shares one bucket despite spoofed XFF", i)
	}
}

// TestTrustedProxyHonorsForwardedFor: when the proxy IS trusted, the
// forwarded client IP keys the bucket.
func TestTrustedProxyHonorsForwardedFor(t *testing.T) {
	// httptest requests arrive from 192.0.2.1 — trust it as the proxy.
	router := routerWith(t, rest.Options{
		RateLimitRPS: 1, RateLimitBurst: 1,
		TrustedProxies: []string{"192.0.2.1"},
	})

	first := doWithHeaders(t, router, http.MethodGet, "/healthz", "",
		map[string]string{"X-Forwarded-For": "203.0.113.7"})
	second := doWithHeaders(t, router, http.MethodGet, "/healthz", "",
		map[string]string{"X-Forwarded-For": "203.0.113.8"})
	assert.Equal(t, http.StatusOK, first.Code)
	assert.Equal(t, http.StatusOK, second.Code, "distinct clients get distinct buckets")
}

// TestInvalidTrustedProxyFallsBackToTrustNone: a bad CIDR must never
// silently widen trust.
func TestInvalidTrustedProxyFallsBackToTrustNone(t *testing.T) {
	router := routerWith(t, rest.Options{
		RateLimitRPS: 1, RateLimitBurst: 1,
		TrustedProxies: []string{"not-a-cidr"},
	})

	first := doWithHeaders(t, router, http.MethodGet, "/healthz", "",
		map[string]string{"X-Forwarded-For": "203.0.113.7"})
	second := doWithHeaders(t, router, http.MethodGet, "/healthz", "",
		map[string]string{"X-Forwarded-For": "203.0.113.8"})
	assert.Equal(t, http.StatusOK, first.Code)
	assert.Equal(t, http.StatusTooManyRequests, second.Code, "spoofed XFF ignored")
}

package rest_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/rest"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales"
	salesdtos "github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/dtos"
	salesservices "github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/services"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses"
	warehousesdtos "github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/dtos"
	warehousesservices "github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/services"
)

// failing query stubs drive the 500 paths of the GET endpoints.
type failingSalesQueries struct{ err error }

func (f *failingSalesQueries) GetSalesOrder(context.Context, string) (salesdtos.SalesOrder, error) {
	return salesdtos.SalesOrder{}, f.err
}

func (f *failingSalesQueries) GetSalesOrders(context.Context, customtypes.Page) ([]salesdtos.SalesOrder, error) {
	return nil, f.err
}

type failingAvailabilityQueries struct{ err error }

func (f *failingAvailabilityQueries) GetAvailability(context.Context, string) (warehousesdtos.Availability, error) {
	return warehousesdtos.Availability{}, f.err
}

func (f *failingAvailabilityQueries) GetAvailabilities(context.Context, customtypes.Page) ([]warehousesdtos.Availability, error) {
	return nil, f.err
}

// newRouter builds the REST layer over facades whose bus is CLOSED, so
// command sends fail — exercising the 500 path without any module wiring.
func newRouter(t *testing.T, ready rest.ReadinessCheck) http.Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	bus := muflone.NewServiceBus(slog.Default())
	require.NoError(t, bus.Close())
	return rest.NewRouter(
		slog.Default(),
		sales.NewFacade(bus, salesservices.NewSalesOrderService()),
		warehouses.NewFacade(bus, warehousesservices.NewAvailabilityService()),
		rest.Options{Ready: ready},
	)
}

func do(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return doWithHeaders(t, handler, method, path, body, nil)
}

func doWithHeaders(t *testing.T, handler http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

func TestBindErrorsReturn400(t *testing.T) {
	router := newRouter(t, nil)

	assert.Equal(t, http.StatusBadRequest,
		do(t, router, http.MethodPost, "/v1/sales", `{"rows": []}`).Code)
	assert.Equal(t, http.StatusBadRequest,
		do(t, router, http.MethodPost, "/v1/warehouses/availability", `{}`).Code)
}

func TestInvalidIdsReturn400ThroughFacadeValidation(t *testing.T) {
	router := newRouter(t, nil)

	salesBody := `{"sales_order_number":"1","customer_name":"x",
		"rows":[{"beer_id":"not-a-uuid","beer_name":"IPA"}]}`
	assert.Equal(t, http.StatusBadRequest,
		do(t, router, http.MethodPost, "/v1/sales", salesBody).Code)

	warehousesBody := `{"beer_id":"not-a-uuid","beer_name":"IPA","quantity":{"value":1,"unit_of_measure":"Lt"}}`
	assert.Equal(t, http.StatusBadRequest,
		do(t, router, http.MethodPost, "/v1/warehouses/availability", warehousesBody).Code)
}

func TestBusFailuresReturn500(t *testing.T) {
	router := newRouter(t, nil)
	beerId := uuid.NewString()

	salesBody := `{"sales_order_number":"1","customer_name":"x",
		"rows":[{"beer_id":"` + beerId + `","beer_name":"IPA","quantity":{"value":1,"unit_of_measure":"Lt"},"price":{"value":5,"currency":"EUR"}}]}`
	assert.Equal(t, http.StatusInternalServerError,
		do(t, router, http.MethodPost, "/v1/sales", salesBody).Code)

	warehousesBody := `{"beer_id":"` + beerId + `","beer_name":"IPA","quantity":{"value":1,"unit_of_measure":"Lt"}}`
	assert.Equal(t, http.StatusInternalServerError,
		do(t, router, http.MethodPost, "/v1/warehouses/availability", warehousesBody).Code)
}

func TestReadModelFailuresReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	bus := muflone.NewServiceBus(slog.Default())
	t.Cleanup(func() { _ = bus.Close() })
	storeErr := errors.New("projection store down")
	router := rest.NewRouter(
		slog.Default(),
		sales.NewFacade(bus, &failingSalesQueries{err: storeErr}),
		warehouses.NewFacade(bus, &failingAvailabilityQueries{err: storeErr}),
		rest.Options{},
	)

	assert.Equal(t, http.StatusInternalServerError,
		do(t, router, http.MethodGet, "/v1/sales", "").Code)
	assert.Equal(t, http.StatusInternalServerError,
		do(t, router, http.MethodGet, "/v1/sales/"+uuid.NewString(), "").Code)
	assert.Equal(t, http.StatusInternalServerError,
		do(t, router, http.MethodGet, "/v1/warehouses/availability", "").Code)
	assert.Equal(t, http.StatusInternalServerError,
		do(t, router, http.MethodGet, "/v1/warehouses/availability/"+uuid.NewString(), "").Code)
}

func TestNotFoundAndEmptyListEndpoints(t *testing.T) {
	router := newRouter(t, nil)

	assert.Equal(t, http.StatusNotFound,
		do(t, router, http.MethodGet, "/v1/sales/"+uuid.NewString(), "").Code)
	assert.Equal(t, http.StatusNotFound,
		do(t, router, http.MethodGet, "/v1/warehouses/availability/"+uuid.NewString(), "").Code)
	assert.Equal(t, http.StatusOK,
		do(t, router, http.MethodGet, "/v1/sales", "").Code)
	assert.Equal(t, http.StatusOK,
		do(t, router, http.MethodGet, "/v1/warehouses/availability", "").Code)
	assert.Equal(t, http.StatusOK,
		do(t, router, http.MethodGet, "/healthz", "").Code)
}

func TestReadinessProbe(t *testing.T) {
	noChecker := newRouter(t, nil)
	assert.Equal(t, http.StatusOK,
		do(t, noChecker, http.MethodGet, "/readyz", "").Code)

	healthy := newRouter(t, func(context.Context) error { return nil })
	assert.Equal(t, http.StatusOK,
		do(t, healthy, http.MethodGet, "/readyz", "").Code)

	unhealthy := newRouter(t, func(context.Context) error { return errors.New("db down") })
	assert.Equal(t, http.StatusServiceUnavailable,
		do(t, unhealthy, http.MethodGet, "/readyz", "").Code)
}

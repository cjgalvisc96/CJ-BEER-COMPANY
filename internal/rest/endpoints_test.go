package rest_test

import (
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
	salesservices "github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/services"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses"
	warehousesservices "github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/services"
)

// newRouter builds the REST layer over facades whose bus is CLOSED, so
// command sends fail — exercising the 500 path without any module wiring.
func newRouter(t *testing.T) http.Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	bus := muflone.NewServiceBus(slog.Default())
	require.NoError(t, bus.Close())
	return rest.NewRouter(
		slog.Default(),
		sales.NewFacade(bus, salesservices.NewSalesOrderService()),
		warehouses.NewFacade(bus, warehousesservices.NewAvailabilityService()),
	)
}

func do(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

func TestBindErrorsReturn400(t *testing.T) {
	router := newRouter(t)

	assert.Equal(t, http.StatusBadRequest,
		do(t, router, http.MethodPost, "/v1/sales", `{"rows": []}`).Code)
	assert.Equal(t, http.StatusBadRequest,
		do(t, router, http.MethodPost, "/v1/warehouses/availability", `{}`).Code)
}

func TestInvalidIdsReturn400ThroughFacadeValidation(t *testing.T) {
	router := newRouter(t)

	salesBody := `{"sales_order_number":"1","customer_name":"x",
		"rows":[{"beer_id":"not-a-uuid","beer_name":"IPA"}]}`
	assert.Equal(t, http.StatusBadRequest,
		do(t, router, http.MethodPost, "/v1/sales", salesBody).Code)

	warehousesBody := `{"beer_id":"not-a-uuid","beer_name":"IPA","quantity":{"value":1,"unit_of_measure":"Lt"}}`
	assert.Equal(t, http.StatusBadRequest,
		do(t, router, http.MethodPost, "/v1/warehouses/availability", warehousesBody).Code)
}

func TestBusFailuresReturn500(t *testing.T) {
	router := newRouter(t)
	beerId := uuid.NewString()

	salesBody := `{"sales_order_number":"1","customer_name":"x",
		"rows":[{"beer_id":"` + beerId + `","beer_name":"IPA","quantity":{"value":1,"unit_of_measure":"Lt"},"price":{"value":5,"currency":"EUR"}}]}`
	assert.Equal(t, http.StatusInternalServerError,
		do(t, router, http.MethodPost, "/v1/sales", salesBody).Code)

	warehousesBody := `{"beer_id":"` + beerId + `","beer_name":"IPA","quantity":{"value":1,"unit_of_measure":"Lt"}}`
	assert.Equal(t, http.StatusInternalServerError,
		do(t, router, http.MethodPost, "/v1/warehouses/availability", warehousesBody).Code)
}

func TestNotFoundAndEmptyListEndpoints(t *testing.T) {
	router := newRouter(t)

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

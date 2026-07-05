package rest_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/auth"
	"github.com/cjgalvisc96/cj-beer-company/internal/rest"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales"
	salesservices "github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/services"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses"
	warehousesservices "github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/services"
)

// fakeVerifier maps tokens to principals — the RBAC middleware is tested
// against the TokenVerifier port, not against Keycloak.
type fakeVerifier struct {
	principals map[string]auth.Principal
}

func (f *fakeVerifier) Verify(_ context.Context, rawToken string) (auth.Principal, error) {
	principal, ok := f.principals[rawToken]
	if !ok {
		return auth.Principal{}, errors.New("unknown token")
	}
	return principal, nil
}

func newAuthedRouter(t *testing.T) http.Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	bus := muflone.NewServiceBus(slog.Default())
	require.NoError(t, bus.Close()) // 500 on writes that pass RBAC — fine here
	verifier := &fakeVerifier{principals: map[string]auth.Principal{
		"manager-token":  auth.NewPrincipal("manager", []string{rest.RoleViewer, rest.RoleSalesManager}),
		"operator-token": auth.NewPrincipal("operator", []string{rest.RoleViewer, rest.RoleWarehouseOperator}),
		"barfly-token":   auth.NewPrincipal("barfly", []string{rest.RoleViewer}),
	}}
	return rest.NewRouter(
		slog.Default(),
		sales.NewFacade(bus, salesservices.NewSalesOrderService()),
		warehouses.NewFacade(bus, warehousesservices.NewAvailabilityService()),
		nil,
		verifier,
	)
}

func authedDo(t *testing.T, handler http.Handler, method, path, token, body string) int {
	t.Helper()
	recorder := doWithHeaders(t, handler, method, path, body, map[string]string{
		"Authorization": "Bearer " + token,
	})
	return recorder.Code
}

func TestRequestsWithoutTokenAre401(t *testing.T) {
	router := newAuthedRouter(t)

	assert.Equal(t, http.StatusUnauthorized,
		do(t, router, http.MethodGet, "/v1/sales", "").Code, "no header")
	assert.Equal(t, http.StatusUnauthorized,
		doWithHeaders(t, router, http.MethodGet, "/v1/sales", "",
			map[string]string{"Authorization": "Basic abc"}).Code, "wrong scheme")
	assert.Equal(t, http.StatusUnauthorized,
		authedDo(t, router, http.MethodGet, "/v1/sales", "forged-token", ""), "invalid token")
}

func TestRBACOnWrites(t *testing.T) {
	router := newAuthedRouter(t)
	salesBody := `{"sales_order_number":"1","customer_name":"x","rows":[]}`
	warehouseBody := `{"beer_id":"x","beer_name":"y","quantity":{"value":1,"unit_of_measure":"Lt"}}`

	// The viewer can read but not write.
	assert.Equal(t, http.StatusOK,
		authedDo(t, router, http.MethodGet, "/v1/sales", "barfly-token", ""))
	assert.Equal(t, http.StatusForbidden,
		authedDo(t, router, http.MethodPost, "/v1/sales", "barfly-token", salesBody))
	assert.Equal(t, http.StatusForbidden,
		authedDo(t, router, http.MethodPost, "/v1/warehouses/availability", "barfly-token", warehouseBody))

	// Roles are not interchangeable.
	assert.Equal(t, http.StatusForbidden,
		authedDo(t, router, http.MethodPost, "/v1/warehouses/availability", "manager-token", warehouseBody))
	assert.Equal(t, http.StatusForbidden,
		authedDo(t, router, http.MethodPost, "/v1/sales", "operator-token", salesBody))

	// The right role passes RBAC and reaches the handler: the sales body
	// is accepted and hits the (closed) bus → 500; the warehouse body has
	// an invalid beer id → 400. Either way, past authorization.
	assert.Equal(t, http.StatusInternalServerError,
		authedDo(t, router, http.MethodPost, "/v1/sales", "manager-token", salesBody))
	assert.Equal(t, http.StatusBadRequest,
		authedDo(t, router, http.MethodPost, "/v1/warehouses/availability", "operator-token", warehouseBody))
}

func TestProbesStayOpen(t *testing.T) {
	router := newAuthedRouter(t)

	assert.Equal(t, http.StatusOK, do(t, router, http.MethodGet, "/healthz", "").Code)
	assert.Equal(t, http.StatusOK, do(t, router, http.MethodGet, "/readyz", "").Code)
}

// Package tests contains black-box end-to-end tests: they drive the HTTP
// API exactly like a client would and assert on the asynchronous outcomes
// of the event choreography (brewing → inventory → orders).
package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/app"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/config"
)

type testClient struct {
	t      *testing.T
	engine http.Handler
}

func newTestClient(t *testing.T) *testClient {
	t.Helper()
	application, err := app.New(config.Config{HTTPAddr: ":0", LogLevel: "error", GinMode: "test"})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	require.NoError(t, application.StartBus(ctx))

	return &testClient{t: t, engine: application.Engine()}
}

func (c *testClient) do(method, path string, body any) (int, map[string]any) {
	c.t.Helper()
	var reader *strings.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(c.t, err)
		reader = strings.NewReader(string(raw))
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	c.engine.ServeHTTP(recorder, req)

	var decoded map[string]any
	if recorder.Body.Len() > 0 {
		require.NoError(c.t, json.Unmarshal(recorder.Body.Bytes(), &decoded),
			"body: %s", recorder.Body.String())
	}
	return recorder.Code, decoded
}

func (c *testClient) createBeer(name string) string {
	c.t.Helper()
	status, body := c.do(http.MethodPost, "/api/v1/beers", map[string]any{
		"name":        name,
		"style":       "ipa",
		"abv":         6.2,
		"price_cents": 550,
		"currency":    "USD",
	})
	require.Equal(c.t, http.StatusCreated, status, "body: %v", body)
	return body["id"].(string)
}

// brewAndStock runs a full production cycle so the beer has sellable stock.
func (c *testClient) brewAndStock(beerID string, units int) {
	c.t.Helper()
	status, batch := c.do(http.MethodPost, "/api/v1/batches", map[string]any{
		"beer_id": beerID,
		"units":   units,
	})
	require.Equal(c.t, http.StatusCreated, status, "body: %v", batch)
	status, _ = c.do(http.MethodPost, "/api/v1/batches/"+batch["id"].(string)+"/complete",
		map[string]any{"produced_units": units})
	require.Equal(c.t, http.StatusOK, status)
}

// eventually polls an assertion until it holds; the choreography is
// asynchronous, so outcomes take a few milliseconds to settle.
func (c *testClient) eventually(check func() bool) {
	c.t.Helper()
	require.Eventually(c.t, check, 3*time.Second, 10*time.Millisecond)
}

func (c *testClient) orderStatus(orderID string) string {
	status, body := c.do(http.MethodGet, "/api/v1/orders/"+orderID, nil)
	require.Equal(c.t, http.StatusOK, status)
	return body["status"].(string)
}

func TestFullHappyPath_BrewStockOrderConfirm(t *testing.T) {
	client := newTestClient(t)

	beerID := client.createBeer("CJ Golden Lager")
	client.brewAndStock(beerID, 100)

	// The BatchCompleted event replenishes inventory asynchronously.
	client.eventually(func() bool {
		status, stock := client.do(http.MethodGet, "/api/v1/stock/"+beerID, nil)
		return status == http.StatusOK && stock["quantity"] == float64(100)
	})

	status, order := client.do(http.MethodPost, "/api/v1/orders", map[string]any{
		"customer_name": "Bar La Cerveceria",
		"lines":         []map[string]any{{"beer_id": beerID, "units": 30}},
	})
	require.Equal(t, http.StatusAccepted, status, "body: %v", order)
	require.Equal(t, "pending", order["status"])
	assert.Equal(t, float64(30*550), order["total_cents"])

	orderID := order["id"].(string)
	client.eventually(func() bool { return client.orderStatus(orderID) == "confirmed" })

	// Stock was reserved: 100 brewed - 30 sold.
	_, stock := client.do(http.MethodGet, "/api/v1/stock/"+beerID, nil)
	assert.Equal(t, float64(70), stock["quantity"])
}

func TestOrderRejectedWhenStockInsufficient(t *testing.T) {
	client := newTestClient(t)

	beerID := client.createBeer("CJ Imperial Stout")
	client.brewAndStock(beerID, 10)
	client.eventually(func() bool {
		status, stock := client.do(http.MethodGet, "/api/v1/stock/"+beerID, nil)
		return status == http.StatusOK && stock["quantity"] == float64(10)
	})

	status, order := client.do(http.MethodPost, "/api/v1/orders", map[string]any{
		"customer_name": "Thirsty Pub",
		"lines":         []map[string]any{{"beer_id": beerID, "units": 999}},
	})
	require.Equal(t, http.StatusAccepted, status)

	orderID := order["id"].(string)
	client.eventually(func() bool { return client.orderStatus(orderID) == "rejected" })

	// Nothing was reserved on a rejected order.
	_, stock := client.do(http.MethodGet, "/api/v1/stock/"+beerID, nil)
	assert.Equal(t, float64(10), stock["quantity"])
}

func TestPlaceOrderForUnknownBeerFailsSynchronously(t *testing.T) {
	client := newTestClient(t)

	status, body := client.do(http.MethodPost, "/api/v1/orders", map[string]any{
		"customer_name": "Ghost Bar",
		"lines": []map[string]any{
			{"beer_id": "00000000-0000-0000-0000-000000000001", "units": 1},
		},
	})

	assert.Equal(t, http.StatusUnprocessableEntity, status, "body: %v", body)
}

func TestRetiredBeerCannotBeOrderedOrBrewed(t *testing.T) {
	client := newTestClient(t)

	beerID := client.createBeer("CJ Seasonal Sour")
	status, _ := client.do(http.MethodDelete, "/api/v1/beers/"+beerID, nil)
	require.Equal(t, http.StatusNoContent, status)

	status, _ = client.do(http.MethodPost, "/api/v1/orders", map[string]any{
		"customer_name": "Bar",
		"lines":         []map[string]any{{"beer_id": beerID, "units": 1}},
	})
	assert.Equal(t, http.StatusUnprocessableEntity, status)

	status, _ = client.do(http.MethodPost, "/api/v1/batches", map[string]any{
		"beer_id": beerID,
		"units":   10,
	})
	assert.Equal(t, http.StatusUnprocessableEntity, status)
}

func TestDuplicateBeerNameConflicts(t *testing.T) {
	client := newTestClient(t)

	client.createBeer("CJ Twin")
	status, _ := client.do(http.MethodPost, "/api/v1/beers", map[string]any{
		"name":        "cj twin",
		"style":       "ale",
		"abv":         4.0,
		"price_cents": 300,
		"currency":    "USD",
	})

	assert.Equal(t, http.StatusConflict, status)
}

func TestHealthEndpoint(t *testing.T) {
	client := newTestClient(t)

	status, body := client.do(http.MethodGet, "/healthz", nil)

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "ok", fmt.Sprint(body["status"]))
}

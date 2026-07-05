// Package tests holds the end-to-end safety net (the book's E2E tests,
// e.g. Can_Create_SalesOrder): drive the HTTP endpoints like a client and
// assert on the read models, accounting for eventual consistency.
package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
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

func (c *testClient) do(method, path string, body any) (int, []byte) {
	c.t.Helper()
	reader := strings.NewReader("")
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(c.t, err)
		reader = strings.NewReader(string(raw))
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	c.engine.ServeHTTP(recorder, req)
	return recorder.Code, recorder.Body.Bytes()
}

func (c *testClient) eventually(check func() bool) {
	c.t.Helper()
	require.Eventually(c.t, check, 3*time.Second, 10*time.Millisecond)
}

// produceBeer declares a finished production order and waits for the
// availability projection.
func (c *testClient) produceBeer(beerId, beerName string, liters int) {
	c.t.Helper()
	status, body := c.do(http.MethodPost, "/v1/warehouses/availability", map[string]any{
		"beer_id":   beerId,
		"beer_name": beerName,
		"quantity":  map[string]any{"value": liters, "unit_of_measure": "Lt"},
	})
	require.Equal(c.t, http.StatusAccepted, status, "body: %s", body)
}

func (c *testClient) availabilityOf(beerId string) (int, bool) {
	status, body := c.do(http.MethodGet, "/v1/warehouses/availability/"+beerId, nil)
	if status != http.StatusOK {
		return 0, false
	}
	var availability struct {
		Quantity struct {
			Value int `json:"value"`
		} `json:"quantity"`
	}
	require.NoError(c.t, json.Unmarshal(body, &availability))
	return availability.Quantity.Value, true
}

// TestCanCreateSalesOrder is the Go rendition of the book's
// Can_Create_SalesOrder E2E test: POST /v1/sales returns Created, and the
// order shows up in the read model.
func TestCanCreateSalesOrder(t *testing.T) {
	client := newTestClient(t)
	beerId := uuid.NewString()

	status, body := client.do(http.MethodPost, "/v1/sales", map[string]any{
		"sales_order_number": "20240315-1500",
		"customer_name":      "Muflone",
		"customer_id":        uuid.NewString(),
		"rows": []map[string]any{{
			"beer_id":   beerId,
			"beer_name": "BrewUp IPA",
			"quantity":  map[string]any{"value": 10, "unit_of_measure": "Lt"},
			"price":     map[string]any{"value": 5, "currency": "EUR"},
		}},
	})
	require.Equal(t, http.StatusCreated, status, "body: %s", body)

	var created struct {
		Id string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(body, &created))
	require.NotEmpty(t, created.Id)

	client.eventually(func() bool {
		status, _ := client.do(http.MethodGet, "/v1/sales/"+created.Id, nil)
		return status == http.StatusOK
	})

	status, body = client.do(http.MethodGet, "/v1/sales/"+created.Id, nil)
	require.Equal(t, http.StatusOK, status)
	var order struct {
		SalesOrderNumber string `json:"sales_order_number"`
		CustomerName     string `json:"customer_name"`
		Rows             []struct {
			BeerName string `json:"beer_name"`
		} `json:"rows"`
	}
	require.NoError(t, json.Unmarshal(body, &order))
	assert.Equal(t, "20240315-1500", order.SalesOrderNumber)
	assert.Equal(t, "Muflone", order.CustomerName)
	require.Len(t, order.Rows, 1)
	assert.Equal(t, "BrewUp IPA", order.Rows[0].BeerName)
}

// TestSalesOrderAllocatesWarehouseStock exercises the book's Figure 4.2
// flow: production fills availability, SalesOrderCreated crosses to the
// warehouse as an integration event, stock is allocated, and
// BeerAvailabilityUpdated leaves the remaining quantity in the read model.
func TestSalesOrderAllocatesWarehouseStock(t *testing.T) {
	client := newTestClient(t)
	beerId := uuid.NewString()

	client.produceBeer(beerId, "BrewUp IPA", 100)
	client.eventually(func() bool {
		quantity, ok := client.availabilityOf(beerId)
		return ok && quantity == 100
	})

	status, _ := client.do(http.MethodPost, "/v1/sales", map[string]any{
		"sales_order_number": "20240315-1501",
		"customer_name":      "Bar La Cerveceria",
		"rows": []map[string]any{{
			"beer_id":   beerId,
			"beer_name": "BrewUp IPA",
			"quantity":  map[string]any{"value": 30, "unit_of_measure": "Lt"},
			"price":     map[string]any{"value": 5, "currency": "EUR"},
		}},
	})
	require.Equal(t, http.StatusCreated, status)

	client.eventually(func() bool {
		quantity, _ := client.availabilityOf(beerId)
		return quantity == 70
	})
}

// allocationStatusOf reads the order's saga outcome from the read model.
func (c *testClient) allocationStatusOf(orderId string) string {
	status, body := c.do(http.MethodGet, "/v1/sales/"+orderId, nil)
	if status != http.StatusOK {
		return ""
	}
	var order struct {
		AllocationStatus string `json:"allocation_status"`
	}
	require.NoError(c.t, json.Unmarshal(body, &order))
	return order.AllocationStatus
}

// TestOversizedOrderIsRejectedAndAvailabilityUntouched: the warehouse
// records QuantityNotFound (book Ch. 12, Fig. 12.3), the saga rejects the
// order with nothing to compensate, and Sales settles the order status.
func TestOversizedOrderIsRejectedAndAvailabilityUntouched(t *testing.T) {
	client := newTestClient(t)
	beerId := uuid.NewString()

	client.produceBeer(beerId, "BrewUp Stout", 10)
	client.eventually(func() bool {
		quantity, ok := client.availabilityOf(beerId)
		return ok && quantity == 10
	})

	status, body := client.do(http.MethodPost, "/v1/sales", map[string]any{
		"sales_order_number": "20240315-1502",
		"customer_name":      "Thirsty Pub",
		"rows": []map[string]any{{
			"beer_id":   beerId,
			"beer_name": "BrewUp Stout",
			"quantity":  map[string]any{"value": 999, "unit_of_measure": "Lt"},
			"price":     map[string]any{"value": 6, "currency": "EUR"},
		}},
	})
	require.Equal(t, http.StatusCreated, status, "the order itself is accepted; allocation is refused downstream")
	var created struct {
		Id string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(body, &created))

	client.eventually(func() bool {
		return client.allocationStatusOf(created.Id) == "rejected"
	})
	quantity, ok := client.availabilityOf(beerId)
	require.True(t, ok)
	assert.Equal(t, 10, quantity)
}

// TestPartialAllocationFailureIsCompensated is the heart of Chapter 12:
// a two-row order allocates the first beer, fails on the second, and the
// saga's compensating transaction gives the first beer's stock back —
// backward recovery, leaving the system consistent.
func TestPartialAllocationFailureIsCompensated(t *testing.T) {
	client := newTestClient(t)
	plentyBeer := uuid.NewString()
	scarceBeer := uuid.NewString()

	client.produceBeer(plentyBeer, "Plenty Lager", 100)
	client.produceBeer(scarceBeer, "Scarce Stout", 5)
	client.eventually(func() bool {
		plenty, okPlenty := client.availabilityOf(plentyBeer)
		scarce, okScarce := client.availabilityOf(scarceBeer)
		return okPlenty && okScarce && plenty == 100 && scarce == 5
	})

	status, body := client.do(http.MethodPost, "/v1/sales", map[string]any{
		"sales_order_number": "20240315-1503",
		"customer_name":      "Greedy Pub",
		"rows": []map[string]any{
			{
				"beer_id":   plentyBeer,
				"beer_name": "Plenty Lager",
				"quantity":  map[string]any{"value": 30, "unit_of_measure": "Lt"},
				"price":     map[string]any{"value": 4, "currency": "EUR"},
			},
			{
				"beer_id":   scarceBeer,
				"beer_name": "Scarce Stout",
				"quantity":  map[string]any{"value": 50, "unit_of_measure": "Lt"},
				"price":     map[string]any{"value": 6, "currency": "EUR"},
			},
		},
	})
	require.Equal(t, http.StatusCreated, status)
	var created struct {
		Id string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(body, &created))

	// The saga rejects the order...
	client.eventually(func() bool {
		return client.allocationStatusOf(created.Id) == "rejected"
	})
	// ...and the compensating transaction restored the first beer's stock.
	client.eventually(func() bool {
		quantity, _ := client.availabilityOf(plentyBeer)
		return quantity == 100
	})
	scarce, _ := client.availabilityOf(scarceBeer)
	assert.Equal(t, 5, scarce, "the failed row never held stock")
}

// TestHappyOrderSettlesAsAllocated: the saga completes and Sales marks the
// order allocated.
func TestHappyOrderSettlesAsAllocated(t *testing.T) {
	client := newTestClient(t)
	beerId := uuid.NewString()

	client.produceBeer(beerId, "BrewUp IPA", 100)
	client.eventually(func() bool {
		quantity, ok := client.availabilityOf(beerId)
		return ok && quantity == 100
	})

	_, body := client.do(http.MethodPost, "/v1/sales", map[string]any{
		"sales_order_number": "20240315-1504",
		"customer_name":      "Happy Bar",
		"rows": []map[string]any{{
			"beer_id":   beerId,
			"beer_name": "BrewUp IPA",
			"quantity":  map[string]any{"value": 30, "unit_of_measure": "Lt"},
			"price":     map[string]any{"value": 5, "currency": "EUR"},
		}},
	})
	var created struct {
		Id string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(body, &created))

	client.eventually(func() bool {
		return client.allocationStatusOf(created.Id) == "allocated"
	})
	quantity, _ := client.availabilityOf(beerId)
	assert.Equal(t, 70, quantity)
}

// TestProductionOrdersAccumulate mirrors the cumulative semantics of the
// book's Example 2: 100 Lt + 100 Lt → 200 Lt.
func TestProductionOrdersAccumulate(t *testing.T) {
	client := newTestClient(t)
	beerId := uuid.NewString()

	client.produceBeer(beerId, "Muflone IPA", 100)
	client.eventually(func() bool {
		quantity, ok := client.availabilityOf(beerId)
		return ok && quantity == 100
	})

	client.produceBeer(beerId, "Muflone IPA", 100)
	client.eventually(func() bool {
		quantity, _ := client.availabilityOf(beerId)
		return quantity == 200
	})
}

func TestInvalidSalesOrderIsRejected(t *testing.T) {
	client := newTestClient(t)

	status, _ := client.do(http.MethodPost, "/v1/sales", map[string]any{
		"customer_name": "No Number",
		"rows":          []map[string]any{},
	})

	assert.Equal(t, http.StatusBadRequest, status)
}

// TestOrderCreationIsIdempotent: a client-supplied id makes retries safe —
// the same POST twice yields one order.
func TestOrderCreationIsIdempotent(t *testing.T) {
	client := newTestClient(t)
	orderId := uuid.NewString()
	body := map[string]any{
		"id":                 orderId,
		"sales_order_number": "retry-0001",
		"customer_name":      "Flaky Network Bar",
		"rows":               []map[string]any{},
	}

	for range 2 {
		status, response := client.do(http.MethodPost, "/v1/sales", body)
		require.Equal(t, http.StatusCreated, status)
		var created struct {
			Id string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(response, &created))
		assert.Equal(t, orderId, created.Id, "the supplied id round-trips")
	}

	client.eventually(func() bool {
		status, _ := client.do(http.MethodGet, "/v1/sales/"+orderId, nil)
		return status == http.StatusOK
	})
	status, response := client.do(http.MethodGet, "/v1/sales?limit=200", nil)
	require.Equal(t, http.StatusOK, status)
	var listing struct {
		Items []struct {
			Id string `json:"id"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(response, &listing))
	occurrences := 0
	for _, item := range listing.Items {
		if item.Id == orderId {
			occurrences++
		}
	}
	assert.Equal(t, 1, occurrences, "no duplicate despite the retry")
}

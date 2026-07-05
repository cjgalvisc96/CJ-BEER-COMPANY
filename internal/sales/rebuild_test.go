package sales_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/dtos"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/services"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel/events"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
)

func seedOrderStream(t *testing.T, store *muflone.InMemoryEventStore, rejected bool) sharedkernel.SalesOrderId {
	t.Helper()
	orderId := sharedkernel.NewSalesOrderId()
	commitId := uuid.New()
	stream := []muflone.DomainEvent{
		events.NewSalesOrderCreated(orderId, commitId,
			sharedkernel.SalesOrderNumber{Value: "20240315-1500"},
			sharedkernel.OrderDate{Value: time.Now().UTC()},
			sharedkernel.CustomerId{Value: uuid.New()},
			sharedkernel.CustomerName{Value: "Muflone"}, nil),
	}
	if rejected {
		stream = append(stream, events.NewSalesOrderAllocationRejected(orderId, commitId, "shortage"))
	} else {
		stream = append(stream, events.NewSalesOrderAllocated(orderId, commitId))
	}
	store.Seed(domain.StreamName+"-"+orderId.Value.String(), stream)
	return orderId
}

// TestRebuildReadModel: the payoff of event sourcing — a fresh read model
// reconstructed entirely from the streams.
func TestRebuildReadModel(t *testing.T) {
	store := muflone.NewInMemoryEventStore()
	allocated := seedOrderStream(t, store, false)
	rejected := seedOrderStream(t, store, true)
	freshReadModel := services.NewSalesOrderService()

	require.NoError(t, sales.RebuildReadModel(context.Background(), store, freshReadModel, slog.Default()))

	ctx := context.Background()
	order, err := freshReadModel.GetSalesOrder(ctx, allocated.Value.String())
	require.NoError(t, err)
	assert.Equal(t, "allocated", order.AllocationStatus)
	order, err = freshReadModel.GetSalesOrder(ctx, rejected.Value.String())
	require.NoError(t, err)
	assert.Equal(t, "rejected", order.AllocationStatus)
	assert.Equal(t, "shortage", order.RejectionReason)

	orders, err := freshReadModel.GetSalesOrders(ctx, customtypes.NewPage(0, 0))
	require.NoError(t, err)
	assert.Len(t, orders, 2)
}

type failingSalesReadModel struct{ services.SalesOrderService }

func (f *failingSalesReadModel) CreateSalesOrder(context.Context, dtos.SalesOrder) error {
	return assert.AnError
}

func TestRebuildSurfacesFailures(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	store := muflone.NewInMemoryEventStore()
	seedOrderStream(t, store, false)
	assert.ErrorIs(t,
		sales.RebuildReadModel(ctx, store, &failingSalesReadModel{}, logger),
		assert.AnError, "projection failure")

	assert.ErrorIs(t,
		sales.RebuildReadModel(ctx, failingSalesStore{}, services.NewSalesOrderService(), logger),
		assert.AnError, "stream listing failure")

	assert.ErrorIs(t,
		sales.RebuildReadModel(ctx, readFailingSalesStore{store}, services.NewSalesOrderService(), logger),
		assert.AnError, "stream read failure")
}

type failingSalesStore struct{ muflone.EventStore }

func (failingSalesStore) ListStreams(context.Context, string) ([]string, error) {
	return nil, assert.AnError
}

type readFailingSalesStore struct{ *muflone.InMemoryEventStore }

func (readFailingSalesStore) ReadStream(context.Context, string) ([]muflone.StoredEvent, error) {
	return nil, assert.AnError
}

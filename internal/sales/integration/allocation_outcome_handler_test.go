package integration_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/integration"
)

func outcomePayload(orderId string) []byte {
	return []byte(`{"commit_id":"` + uuid.NewString() + `","sales_order_id":"` + orderId + `","reason":"shortage"}`)
}

func TestOutcomeHandlersSendSettlementCommands(t *testing.T) {
	bus := muflone.NewServiceBus(slog.Default())
	t.Cleanup(func() { _ = bus.Close() })
	handler := integration.NewAllocationOutcomeHandler(bus, slog.Default())
	ctx := context.Background()

	assert.NoError(t, handler.OnAllocationCompleted(ctx, outcomePayload(uuid.NewString())))
	assert.NoError(t, handler.OnAllocationRejected(ctx, outcomePayload(uuid.NewString())))
}

func TestOutcomeHandlersRejectMalformedPayloads(t *testing.T) {
	bus := muflone.NewServiceBus(slog.Default())
	t.Cleanup(func() { _ = bus.Close() })
	handler := integration.NewAllocationOutcomeHandler(bus, slog.Default())
	ctx := context.Background()

	assert.Error(t, handler.OnAllocationCompleted(ctx, []byte(`not json`)))
	assert.Error(t, handler.OnAllocationRejected(ctx, []byte(`not json`)))
}

func TestOutcomeHandlersIgnoreUnusableOrderIds(t *testing.T) {
	bus := muflone.NewServiceBus(slog.Default())
	t.Cleanup(func() { _ = bus.Close() })
	handler := integration.NewAllocationOutcomeHandler(bus, slog.Default())
	ctx := context.Background()

	assert.NoError(t, handler.OnAllocationCompleted(ctx, outcomePayload("not-a-uuid")))
	assert.NoError(t, handler.OnAllocationRejected(ctx, outcomePayload("not-a-uuid")))
}

func TestOutcomeHandlersSurfaceBusFailures(t *testing.T) {
	bus := muflone.NewServiceBus(slog.Default())
	require.NoError(t, bus.Close())
	handler := integration.NewAllocationOutcomeHandler(bus, slog.Default())
	ctx := context.Background()

	assert.Error(t, handler.OnAllocationCompleted(ctx, outcomePayload(uuid.NewString())))
	assert.Error(t, handler.OnAllocationRejected(ctx, outcomePayload(uuid.NewString())))
}

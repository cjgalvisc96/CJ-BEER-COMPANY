package eventhandlers

import (
	"context"
	"log/slog"

	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel/events"
)

// allocationStatusWriter is the slice of the read-model service the status
// projections need.
type allocationStatusWriter interface {
	UpdateAllocationStatus(ctx context.Context, salesOrderId, status, reason string) error
}

// AllocationStatusProjector keeps the order projection's allocation status
// in sync with the aggregate's settlement events.
type AllocationStatusProjector struct {
	service allocationStatusWriter
	logger  *slog.Logger
}

func NewAllocationStatusProjector(service allocationStatusWriter, logger *slog.Logger) *AllocationStatusProjector {
	return &AllocationStatusProjector{service: service, logger: logger}
}

func (p *AllocationStatusProjector) OnSalesOrderAllocated(ctx context.Context, event events.SalesOrderAllocated) error {
	return p.service.UpdateAllocationStatus(ctx, event.SalesOrderId.Value.String(), "allocated", "")
}

func (p *AllocationStatusProjector) OnSalesOrderAllocationRejected(
	ctx context.Context,
	event events.SalesOrderAllocationRejected,
) error {
	return p.service.UpdateAllocationStatus(ctx, event.SalesOrderId.Value.String(), "rejected", event.Reason)
}

// Command rebuild reconstructs every read-model projection from the event
// store — the operational payoff of event sourcing (recover a corrupted
// read model, or populate a brand-new projection). Durable mode only:
// DB_URL must point at the Postgres holding the streams.
//
//	DB_URL=postgres://... go run ./cmd/rebuild   (or: task rebuild)
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/config"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/database"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/logging"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales"
	salesservices "github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/services"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses"
	warehousesservices "github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/services"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "rebuild failed:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	if cfg.DBURL == "" {
		return fmt.Errorf("DB_URL must be set — there is nothing to rebuild in memory")
	}
	logger := logging.New(cfg.LogLevel)
	ctx := context.Background()

	db, err := database.Open(cfg.DBURL)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	salesReadModel := salesservices.NewPostgresSalesOrderService(db)
	if err := salesReadModel.Reset(ctx); err != nil {
		return err
	}
	if err := sales.RebuildReadModel(ctx,
		muflone.NewPostgresEventStore(db, sales.NewEventRegistry()), salesReadModel, logger); err != nil {
		return err
	}

	availabilityReadModel := warehousesservices.NewPostgresAvailabilityService(db)
	if err := availabilityReadModel.Reset(ctx); err != nil {
		return err
	}
	if err := warehouses.RebuildReadModel(ctx,
		muflone.NewPostgresEventStore(db, warehouses.NewEventRegistry()), availabilityReadModel, logger); err != nil {
		return err
	}

	logger.Info("rebuild.completed")
	return nil
}

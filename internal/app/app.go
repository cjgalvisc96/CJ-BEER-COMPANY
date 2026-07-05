// Package app is the composition root: it builds the DI container, wires
// every bounded context, subscribes event handlers to the bus, and runs the
// HTTP server and message router with graceful shutdown.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	"github.com/cjgalvisc96/cj-beer-company/internal/brewing"
	brewingcommands "github.com/cjgalvisc96/cj-beer-company/internal/brewing/application/commands"
	brewingqueries "github.com/cjgalvisc96/cj-beer-company/internal/brewing/application/queries"
	"github.com/cjgalvisc96/cj-beer-company/internal/catalog"
	catalogcommands "github.com/cjgalvisc96/cj-beer-company/internal/catalog/application/commands"
	catalogqueries "github.com/cjgalvisc96/cj-beer-company/internal/catalog/application/queries"
	"github.com/cjgalvisc96/cj-beer-company/internal/inventory"
	inventorycommands "github.com/cjgalvisc96/cj-beer-company/internal/inventory/application/commands"
	inventoryqueries "github.com/cjgalvisc96/cj-beer-company/internal/inventory/application/queries"
	"github.com/cjgalvisc96/cj-beer-company/internal/orders"
	orderscommands "github.com/cjgalvisc96/cj-beer-company/internal/orders/application/commands"
	ordersqueries "github.com/cjgalvisc96/cj-beer-company/internal/orders/application/queries"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/config"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/logging"
	httppresentation "github.com/cjgalvisc96/cj-beer-company/internal/presentation/http"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/application/ports"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/infrastructure/messaging"
)

const shutdownTimeout = 10 * time.Second

type App struct {
	Injector do.Injector
	cfg      config.Config
	logger   *slog.Logger
	bus      *messaging.Bus
	engine   *gin.Engine
}

// New wires the whole application. It only fails on wiring errors, so a
// successful New means the app is ready to Run.
func New(cfg config.Config) (*App, error) {
	logger := logging.New(cfg.LogLevel)
	gin.SetMode(cfg.GinMode)

	injector := do.New()
	do.ProvideValue(injector, cfg)
	do.ProvideValue(injector, logger)

	bus, err := messaging.NewBus(logger)
	if err != nil {
		return nil, fmt.Errorf("create message bus: %w", err)
	}
	do.ProvideValue(injector, bus)
	do.Provide(injector, func(i do.Injector) (ports.EventPublisher, error) {
		return messaging.NewWatermillEventPublisher(bus.PubSub), nil
	})

	// Bounded contexts. Registration order does not matter (providers are
	// lazy); subscription order is explicit below.
	catalog.Register(injector)
	inventory.Register(injector)
	orders.Register(injector)
	brewing.Register(injector)

	inventory.SubscribeEventHandlers(injector, bus)
	orders.SubscribeEventHandlers(injector, bus)

	engine := httppresentation.NewRouter(
		logger,
		httppresentation.NewBeerHandlers(
			do.MustInvoke[*catalogcommands.CreateBeerHandler](injector),
			do.MustInvoke[*catalogcommands.ChangeBeerPriceHandler](injector),
			do.MustInvoke[*catalogcommands.RetireBeerHandler](injector),
			do.MustInvoke[*catalogqueries.GetBeerHandler](injector),
			do.MustInvoke[*catalogqueries.ListBeersHandler](injector),
		),
		httppresentation.NewStockHandlers(
			do.MustInvoke[*inventorycommands.TrackStockItemHandler](injector),
			do.MustInvoke[*inventorycommands.ReplenishStockHandler](injector),
			do.MustInvoke[*inventoryqueries.GetStockHandler](injector),
			do.MustInvoke[*inventoryqueries.ListStockHandler](injector),
		),
		httppresentation.NewOrderHandlers(
			do.MustInvoke[*orderscommands.PlaceOrderHandler](injector),
			do.MustInvoke[*orderscommands.CancelOrderHandler](injector),
			do.MustInvoke[*ordersqueries.GetOrderHandler](injector),
			do.MustInvoke[*ordersqueries.ListOrdersHandler](injector),
		),
		httppresentation.NewBatchHandlers(
			do.MustInvoke[*brewingcommands.StartBatchHandler](injector),
			do.MustInvoke[*brewingcommands.CompleteBatchHandler](injector),
			do.MustInvoke[*brewingqueries.GetBatchHandler](injector),
			do.MustInvoke[*brewingqueries.ListBatchesHandler](injector),
		),
	)

	return &App{Injector: injector, cfg: cfg, logger: logger, bus: bus, engine: engine}, nil
}

// Run blocks until ctx is cancelled, then shuts everything down gracefully.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 2)

	go func() {
		if err := a.bus.Router.Run(ctx); err != nil {
			errCh <- fmt.Errorf("message router: %w", err)
		}
	}()
	<-a.bus.Router.Running()

	server := &nethttp.Server{Addr: a.cfg.HTTPAddr, Handler: a.engine}
	go func() {
		a.logger.Info("http.listening", slog.String("addr", a.cfg.HTTPAddr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			errCh <- fmt.Errorf("http server: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		a.logger.Info("app.shutting_down")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	if err := a.bus.Close(); err != nil {
		return fmt.Errorf("bus shutdown: %w", err)
	}
	return nil
}

// Engine exposes the HTTP handler for tests.
func (a *App) Engine() *gin.Engine {
	return a.engine
}

// StartBus runs only the message router (used by end-to-end tests that
// drive the Engine directly instead of a real listener).
func (a *App) StartBus(ctx context.Context) error {
	go func() {
		_ = a.bus.Router.Run(ctx)
	}()
	select {
	case <-a.bus.Router.Running():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

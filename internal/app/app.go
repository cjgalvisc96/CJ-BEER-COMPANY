// Package app is the composition root: it builds the DI container, the
// service bus, wires the Sales and Warehouses modules, and runs the HTTP
// server and the bus with graceful shutdown.
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

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/config"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/logging"
	"github.com/cjgalvisc96/cj-beer-company/internal/rest"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses"
)

const shutdownTimeout = 10 * time.Second

type App struct {
	Injector do.Injector
	cfg      config.Config
	logger   *slog.Logger
	bus      *muflone.ServiceBus
	engine   *gin.Engine
}

func New(cfg config.Config) (*App, error) {
	logger := logging.New(cfg.LogLevel)
	gin.SetMode(cfg.GinMode)

	injector := do.New()
	do.ProvideValue(injector, cfg)
	do.ProvideValue(injector, logger)

	bus := muflone.NewServiceBus(logger)

	sales.Register(injector, bus)
	warehouses.Register(injector, bus)

	engine := rest.NewRouter(
		logger,
		do.MustInvoke[*sales.Facade](injector),
		do.MustInvoke[*warehouses.Facade](injector),
	)

	return &App{Injector: injector, cfg: cfg, logger: logger, bus: bus, engine: engine}, nil
}

// Run blocks until ctx is cancelled, then shuts down gracefully.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 2)

	go func() {
		if err := a.bus.Run(ctx); err != nil {
			errCh <- fmt.Errorf("service bus: %w", err)
		}
	}()
	<-a.bus.Running()

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

// StartBus runs only the service bus, for tests that drive Engine directly.
func (a *App) StartBus(ctx context.Context) error {
	go func() {
		_ = a.bus.Run(ctx)
	}()
	select {
	case <-a.bus.Running():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

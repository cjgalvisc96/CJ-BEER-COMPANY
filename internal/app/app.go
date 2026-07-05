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
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/auth"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/config"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/database"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/logging"
	"github.com/cjgalvisc96/cj-beer-company/internal/rest"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sagas"
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

	// Durable mode: one shared Postgres pool, fail fast if unreachable.
	var ready rest.ReadinessCheck
	if cfg.DBURL != "" {
		db, err := database.Open(cfg.DBURL)
		if err != nil {
			return nil, err
		}
		do.ProvideValue(injector, db)
		ready = db.PingContext
		logger.Info("persistence.durable", slog.String("driver", "postgres"))
	} else {
		logger.Info("persistence.in_memory")
	}

	// Broker mode: RabbitMQ on the wire (the book's transport), so
	// messages survive restarts too; otherwise the in-process GoChannel.
	var bus *muflone.ServiceBus
	if cfg.BrokerURL != "" {
		transport, err := muflone.NewAMQPTransport(cfg.BrokerURL, logger)
		if err != nil {
			return nil, err
		}
		bus = muflone.NewServiceBusWithTransport(transport, logger)
		logger.Info("messaging.broker", slog.String("transport", "amqp"))
	} else {
		bus = muflone.NewServiceBus(logger)
		logger.Info("messaging.in_memory")
	}

	sales.Register(injector, bus)
	warehouses.Register(injector, bus)

	// Auth mode: OIDC bearer tokens + RBAC when an issuer is configured
	// (Keycloak in the compose stack); open API otherwise (dev, tests).
	var verifier rest.TokenVerifier
	if cfg.AuthIssuer != "" {
		verifier = auth.NewOIDCVerifier(context.Background(), cfg.AuthIssuer, cfg.AuthJWKSURL, cfg.AuthClientID)
		logger.Info("auth.oidc", slog.String("issuer", cfg.AuthIssuer))
	} else {
		logger.Warn("auth.disabled")
	}

	engine := rest.NewRouter(
		logger,
		do.MustInvoke[*sales.Facade](injector),
		do.MustInvoke[*warehouses.Facade](injector),
		ready,
		verifier,
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
	a.startSagaSupervision(ctx)

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

// startSagaSupervision implements the durable-execution duties of the
// composition root (ADR-0008): resume in-flight sagas at boot, then watch
// for timed-out steps.
func (a *App) startSagaSupervision(ctx context.Context) {
	saga := do.MustInvoke[*sagas.OrderAllocationSaga](a.Injector)
	if err := saga.ResumeInFlight(ctx); err != nil {
		a.logger.Error("saga.resume_failed", slog.String("error", err.Error()))
	}
	if a.cfg.SagaStepTimeout <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(a.cfg.SagaStepTimeout / 2)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := saga.TimeoutInFlight(ctx, time.Now().Add(-a.cfg.SagaStepTimeout)); err != nil {
					a.logger.Error("saga.timeout_sweep_failed", slog.String("error", err.Error()))
				}
			}
		}
	}()
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

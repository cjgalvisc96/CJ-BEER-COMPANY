// Package app is the composition root: it builds the DI container, the
// service bus, wires the Sales and Warehouses modules, and runs the HTTP
// server and the bus with graceful shutdown.
package app

import (
	"context"
	"database/sql"
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
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/telemetry"
	"github.com/cjgalvisc96/cj-beer-company/internal/rest"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sagas"
)

const shutdownTimeout = 10 * time.Second

type App struct {
	Injector        do.Injector
	cfg             config.Config
	logger          *slog.Logger
	bus             *muflone.ServiceBus
	engine          *gin.Engine
	shutdownTracing func(context.Context) error
	relay           *muflone.OutboxRelay
}

func New(cfg config.Config) (*App, error) {
	// Tag every log line with the environment so a shared aggregator can
	// tell deployments apart, not just the boot banner.
	logger := logging.New(cfg.LogLevel).With(slog.String("env", cfg.AppEnv))
	logger.Info("app.environment")
	if !cfg.EnvironmentRecognized() {
		logger.Warn("app.environment.unrecognized", slog.Any("expected", config.KnownEnvironments))
	}
	gin.SetMode(cfg.GinMode)

	injector := do.New()
	do.ProvideValue(injector, cfg)
	do.ProvideValue(injector, logger)

	// Durable mode: one shared Postgres pool, fail fast if unreachable.
	var ready rest.ReadinessCheck
	var db *sql.DB
	if cfg.DBURL != "" {
		opened, err := database.Open(cfg.DBURL)
		if err != nil {
			return nil, err
		}
		db = opened
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

	// Durable mode: the transactional-outbox relay is the publisher, and
	// poison messages are archived for `task redrive`.
	var relay *muflone.OutboxRelay
	if db != nil {
		relay = muflone.NewOutboxRelay(db, bus, cfg.OutboxInterval, logger)
		deadLetters := muflone.NewDeadLetterStore(db)
		bus.OnDeadLetter("muflone.dead_letter_archive", deadLetters.Save)
		logger.Info("messaging.outbox", slog.Duration("interval", cfg.OutboxInterval))
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

	// Observability: Prometheus metrics always (cheap, local); OTLP
	// traces when an endpoint is configured.
	shutdownTracing := telemetry.InitTracing(context.Background(), cfg.OTELEndpoint, cfg.ServiceName)
	metricsHandler, err := telemetry.InitMetrics(cfg.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("init metrics: %w", err)
	}

	engine := rest.NewRouter(
		logger,
		do.MustInvoke[*sales.Facade](injector),
		do.MustInvoke[*warehouses.Facade](injector),
		rest.Options{
			Ready:          ready,
			Verifier:       verifier,
			MaxBodyBytes:   cfg.MaxBodyBytes,
			RateLimitRPS:   cfg.RateLimitRPS,
			RateLimitBurst: cfg.RateLimitBurst,
			MetricsHandler: metricsHandler,
			TracingEnabled: cfg.OTELEndpoint != "",
			TrustedProxies: cfg.TrustedProxies,
		},
	)

	return &App{
		Injector: injector, cfg: cfg, logger: logger, bus: bus, engine: engine,
		shutdownTracing: shutdownTracing, relay: relay,
	}, nil
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
	if a.relay != nil {
		go a.relay.Run(ctx)
	}
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
	if err := a.shutdownTracing(shutdownCtx); err != nil {
		return fmt.Errorf("tracing shutdown: %w", err)
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

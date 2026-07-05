package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/cjgalvisc96/cj-beer-company/internal/platform/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DB_URL", "")
	t.Setenv("BROKER_URL", "")
	cfg := config.Load()

	assert.Equal(t, ":8080", cfg.HTTPAddr)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "release", cfg.GinMode)
	assert.Empty(t, cfg.DBURL, "in-memory persistence by default")
	assert.Empty(t, cfg.BrokerURL, "in-process messaging by default")
	assert.Equal(t, 5*time.Minute, cfg.SagaStepTimeout)
}

func TestBrokerURLFromEnvironment(t *testing.T) {
	t.Setenv("BROKER_URL", "amqp://guest:guest@localhost:5672/")
	assert.Equal(t, "amqp://guest:guest@localhost:5672/", config.Load().BrokerURL)
}

func TestAuthConfig(t *testing.T) {
	t.Setenv("AUTH_ISSUER", "")
	t.Setenv("AUTH_JWKS_URL", "")
	cfg := config.Load()
	assert.Empty(t, cfg.AuthIssuer, "auth disabled by default")
	assert.Empty(t, cfg.AuthJWKSURL, "no JWKS derived while disabled")
	assert.Equal(t, "brewup-api", cfg.AuthClientID)

	t.Setenv("AUTH_ISSUER", "http://localhost:8180/realms/brewup")
	cfg = config.Load()
	assert.Equal(t, "http://localhost:8180/realms/brewup/protocol/openid-connect/certs",
		cfg.AuthJWKSURL, "JWKS defaults to the Keycloak convention under the issuer")

	t.Setenv("AUTH_JWKS_URL", "http://keycloak:8080/realms/brewup/protocol/openid-connect/certs")
	t.Setenv("AUTH_CLIENT_ID", "other-client")
	cfg = config.Load()
	assert.Equal(t, "http://keycloak:8080/realms/brewup/protocol/openid-connect/certs", cfg.AuthJWKSURL)
	assert.Equal(t, "other-client", cfg.AuthClientID)
}

func TestSagaStepTimeoutParsing(t *testing.T) {
	t.Setenv("SAGA_STEP_TIMEOUT", "90s")
	assert.Equal(t, 90*time.Second, config.Load().SagaStepTimeout)

	t.Setenv("SAGA_STEP_TIMEOUT", "not-a-duration")
	assert.Equal(t, 5*time.Minute, config.Load().SagaStepTimeout, "unparseable falls back")

	t.Setenv("SAGA_STEP_TIMEOUT", "")
	assert.Equal(t, 5*time.Minute, config.Load().SagaStepTimeout, "empty falls back")

	t.Setenv("SAGA_STEP_TIMEOUT", "0")
	assert.Equal(t, time.Duration(0), config.Load().SagaStepTimeout, "0 disables the watchdog")
}

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":9999")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("GIN_MODE", "") // empty falls back to the default
	t.Setenv("DB_URL", "postgres://beer@localhost/beer")

	cfg := config.Load()

	assert.Equal(t, ":9999", cfg.HTTPAddr)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "release", cfg.GinMode)
	assert.Equal(t, "postgres://beer@localhost/beer", cfg.DBURL)
}

func TestHardeningAndTelemetryConfig(t *testing.T) {
	cfg := config.Load()
	assert.Equal(t, "cj-beer-company", cfg.ServiceName)
	assert.Equal(t, 50.0, cfg.RateLimitRPS)
	assert.Equal(t, 100, cfg.RateLimitBurst)
	assert.Equal(t, int64(1<<20), cfg.MaxBodyBytes)
	assert.Empty(t, cfg.OTELEndpoint)

	t.Setenv("RATE_LIMIT_RPS", "2.5")
	t.Setenv("RATE_LIMIT_BURST", "7")
	t.Setenv("MAX_BODY_BYTES", "1024")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4318")
	cfg = config.Load()
	assert.Equal(t, 2.5, cfg.RateLimitRPS)
	assert.Equal(t, 7, cfg.RateLimitBurst)
	assert.Equal(t, int64(1024), cfg.MaxBodyBytes)
	assert.Equal(t, "http://collector:4318", cfg.OTELEndpoint)

	t.Setenv("RATE_LIMIT_RPS", "junk")
	t.Setenv("RATE_LIMIT_BURST", "junk")
	cfg = config.Load()
	assert.Equal(t, 50.0, cfg.RateLimitRPS, "unparseable falls back")
	assert.Equal(t, 100, cfg.RateLimitBurst, "unparseable falls back")
}

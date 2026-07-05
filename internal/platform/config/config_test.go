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

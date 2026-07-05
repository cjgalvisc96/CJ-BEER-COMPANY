package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cjgalvisc96/cj-beer-company/internal/platform/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DB_URL", "")
	cfg := config.Load()

	assert.Equal(t, ":8080", cfg.HTTPAddr)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "release", cfg.GinMode)
	assert.Empty(t, cfg.DBURL, "in-memory persistence by default")
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

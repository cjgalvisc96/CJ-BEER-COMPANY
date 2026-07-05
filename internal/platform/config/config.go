// Package config centralizes environment-driven configuration.
package config

import (
	"os"
	"time"
)

type Config struct {
	HTTPAddr string
	LogLevel string
	GinMode  string
	// DBURL switches persistence: empty runs everything in memory (dev,
	// tests); a Postgres URL makes the event store and the read models
	// durable (production).
	DBURL string
	// BrokerURL switches the service-bus transport: empty runs the
	// in-process GoChannel (dev, tests); an AMQP URL puts RabbitMQ on the
	// wire (production — messages survive restarts too).
	BrokerURL string
	// SagaStepTimeout fails a saga step that made no progress for this
	// long (0 disables the watchdog).
	SagaStepTimeout time.Duration
	// AuthIssuer switches authentication: empty leaves the API open
	// (dev, tests); an OIDC issuer URL turns on bearer-token
	// authentication and RBAC. It must equal the tokens' `iss` claim.
	AuthIssuer string
	// AuthJWKSURL is where the signing keys are fetched; defaults to the
	// Keycloak convention under the issuer. Configured separately so the
	// issuer can be host-visible while keys travel the container network.
	AuthJWKSURL string
	// AuthClientID is the expected audience of the tokens.
	AuthClientID string
}

// Load reads configuration from the environment, falling back to sane
// development defaults.
func Load() Config {
	cfg := Config{
		HTTPAddr:        getEnv("HTTP_ADDR", ":8080"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		GinMode:         getEnv("GIN_MODE", "release"),
		DBURL:           getEnv("DB_URL", ""),
		BrokerURL:       getEnv("BROKER_URL", ""),
		SagaStepTimeout: getDuration("SAGA_STEP_TIMEOUT", 5*time.Minute),
		AuthIssuer:      getEnv("AUTH_ISSUER", ""),
		AuthJWKSURL:     getEnv("AUTH_JWKS_URL", ""),
		AuthClientID:    getEnv("AUTH_CLIENT_ID", "brewup-api"),
	}
	if cfg.AuthJWKSURL == "" && cfg.AuthIssuer != "" {
		cfg.AuthJWKSURL = cfg.AuthIssuer + "/protocol/openid-connect/certs"
	}
	return cfg
}

func getDuration(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

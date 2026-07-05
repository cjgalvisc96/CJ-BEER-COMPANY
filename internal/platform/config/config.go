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
}

// Load reads configuration from the environment, falling back to sane
// development defaults.
func Load() Config {
	return Config{
		HTTPAddr:        getEnv("HTTP_ADDR", ":8080"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		GinMode:         getEnv("GIN_MODE", "release"),
		DBURL:           getEnv("DB_URL", ""),
		BrokerURL:       getEnv("BROKER_URL", ""),
		SagaStepTimeout: getDuration("SAGA_STEP_TIMEOUT", 5*time.Minute),
	}
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

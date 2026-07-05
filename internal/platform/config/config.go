// Package config centralizes environment-driven configuration.
package config

import (
	"os"
)

type Config struct {
	HTTPAddr string
	LogLevel string
	GinMode  string
}

// Load reads configuration from the environment, falling back to sane
// development defaults.
func Load() Config {
	return Config{
		HTTPAddr: getEnv("HTTP_ADDR", ":8080"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
		GinMode:  getEnv("GIN_MODE", "release"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

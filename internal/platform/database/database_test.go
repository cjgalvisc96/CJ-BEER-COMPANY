package database_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cjgalvisc96/cj-beer-company/internal/platform/database"
)

func TestOpenFailsFastWhenUnreachable(t *testing.T) {
	// Nothing listens on port 1; the ping must fail quickly.
	_, err := database.Open("postgres://beer:beer@127.0.0.1:1/beer?sslmode=disable&connect_timeout=1")

	assert.ErrorContains(t, err, "ping database")
}

func TestOpenRejectsMalformedURL(t *testing.T) {
	_, err := database.Open("://not-a-dsn")

	assert.Error(t, err)
}

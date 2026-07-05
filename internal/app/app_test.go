// Smoke tests for the composition root: the app starts, serves, and shuts
// down gracefully. internal/app is exempt from the 100% gate (its
// remaining branches are runtime fault-injection paths), but the happy
// path and the failure-to-listen path are still proven here.
package app_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/app"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/config"
)

func TestRunServesAndShutsDownGracefully(t *testing.T) {
	application, err := app.New(config.Config{
		HTTPAddr: "127.0.0.1:18191", LogLevel: "error", GinMode: "test",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- application.Run(ctx) }()

	require.Eventually(t, func() bool {
		response, err := http.Get("http://127.0.0.1:18191/healthz")
		if err != nil {
			return false
		}
		defer response.Body.Close()
		return response.StatusCode == http.StatusOK
	}, 3*time.Second, 25*time.Millisecond)

	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("app did not shut down")
	}
}

func TestRunFailsWhenItCannotListen(t *testing.T) {
	application, err := app.New(config.Config{
		HTTPAddr: "not-a-valid-listen-address", LogLevel: "error", GinMode: "test",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	assert.Error(t, application.Run(ctx))
}

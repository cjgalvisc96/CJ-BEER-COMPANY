package logging_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/platform/logging"
)

func TestNewHonorsLevel(t *testing.T) {
	cases := []struct {
		level   string
		enabled slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"info", slog.LevelInfo},
		{"unknown-defaults-to-info", slog.LevelInfo},
	}
	for _, testCase := range cases {
		logger := logging.New(testCase.level)
		require.NotNil(t, logger)
		assert.True(t, logger.Enabled(context.Background(), testCase.enabled),
			"level %q must enable %v", testCase.level, testCase.enabled)
	}
}

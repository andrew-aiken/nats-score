package logging_test

import (
	"context"
	"log/slog"
	"testing"

	"server/pkg/logging"
)

func TestLogger(t *testing.T) {
	tests := []struct {
		name          string
		logLevel      string
		expectedLevel slog.Level
		pass          bool
	}{
		{
			name:          "debug",
			logLevel:      "debug",
			expectedLevel: slog.LevelDebug,
			pass:          true,
		},
		{
			name:          "info",
			logLevel:      "info",
			expectedLevel: slog.LevelInfo,
			pass:          true,
		},
		{
			name:          "warn",
			logLevel:      "warn",
			expectedLevel: slog.LevelWarn,
			pass:          true,
		},
		{
			name:          "error",
			logLevel:      "error",
			expectedLevel: slog.LevelError,
			pass:          true,
		},
		{
			name:          "default",
			logLevel:      "dne",
			expectedLevel: slog.LevelInfo,
			pass:          true,
		},
		{
			name:          "wrongExpected",
			logLevel:      "warn",
			expectedLevel: slog.LevelDebug,
			pass:          false,
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logging.SetupLogging(tt.logLevel)
			if slog.Default().Enabled(ctx, tt.expectedLevel) != tt.pass {
				t.Errorf("Result does not match, expected %s", tt.logLevel)
			}
		})
	}

	// Mainly for coverage
	t.Run("replaceLogAttribute", func(t *testing.T) {
		logging.SetupLogging("info")
		slog.Info("Test slog message")
	})
}

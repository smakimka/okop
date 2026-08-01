package logger

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

var instance *slog.Logger

func init() {
	Configure(os.Stderr, slog.LevelDebug)
}

// Configure rebuilds the global logger to write to w at the given level.
// Callers that render their own output — see internal/ui — pass a writer that
// keeps log lines from colliding with what is already on screen.
func Configure(w io.Writer, level slog.Level) {
	instance = slog.New(tint.NewHandler(w, &tint.Options{
		Level:      level,
		TimeFormat: time.Kitchen,
	}))

	slog.SetDefault(instance)
}

func Global() *slog.Logger {
	return instance
}

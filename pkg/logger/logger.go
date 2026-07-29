package logger

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

var instance *slog.Logger

func init() {
	instance = slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.Kitchen,
	}))

	slog.SetDefault(instance)
}

func Global() *slog.Logger {
	return instance
}

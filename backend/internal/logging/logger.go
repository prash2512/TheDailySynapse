package logging

import (
	"log/slog"
	"os"
)

func New(level string) *slog.Logger {
	var lvl slog.Level
	lvl.UnmarshalText([]byte(level))

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	}))
}

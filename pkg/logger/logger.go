package logger

import (
	"log/slog"
	"os"
)

func GetLogger() *slog.Logger {
	loggerOpts := slog.HandlerOptions{}
	return slog.New(slog.NewTextHandler(os.Stdout, &loggerOpts))
}

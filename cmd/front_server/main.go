package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"simple-s3-adventure/internal/front_server"
	"syscall"
)

var (
	port = ":13090"
)

func main() {
	loggerOpts := slog.HandlerOptions{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &loggerOpts))

	ctx, cancel := context.WithCancelCause(context.Background())
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		logger.Debug("terminate front server")
		cancel(errors.New("app termination by sigterm"))
	}()

	front_server.StartServer(ctx, logger, port)
}

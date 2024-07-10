package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"simple-s3-adventure/internal/chunk_server/service"
	"time"

	"github.com/cenkalti/backoff"

	"simple-s3-adventure/pkg/logger"
)

const (
	registrationDelay = 5 * time.Second
)

func registerHandlers(mux *http.ServeMux, config *service.ServerConfig) {
	mux.HandleFunc("/put", func(w http.ResponseWriter, r *http.Request) {
		PutHandler(w, r, config)
	})
	mux.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		GetHandler(w, r, config)
	})
	mux.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
		DeleteHandler(w, r, config)
	})
}

// StartServer starts the HTTP server on the given port.
func StartServer(config *service.ServerConfig) {
	mux := http.NewServeMux()
	registerHandlers(mux, config)

	lg := logger.GetLogger()

	// Wait for launch of HTTP server
	time.AfterFunc(registrationDelay, func() {
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(fmt.Errorf("registration cancelled"))

		var attempt int
		bo := backoff.WithContext(backoff.NewExponentialBackOff(), ctx)
		if err := backoff.Retry(func() error {
			attempt++
			if err := register(config.FrontServerAddress, config.Port); err != nil {
				lg.Error("Failed to register chunk server", slog.Int("attempt", attempt), slog.String("error", err.Error()))
				return err
			}
			return nil
		}, bo); err != nil {
			if errors.Is(err, context.Canceled) {
				lg.Info("Registration cancelled")
			} else {
				lg.Error("Failed to register chunk server", slog.String("error", err.Error()))
			}
			os.Exit(1)
		}
	})

	lg.Info("Starting chunk server", slog.String("port", config.Port))
	address := net.JoinHostPort("", config.Port)
	if err := http.ListenAndServe(address, mux); err != nil {
		lg.Error("Could not start server", slog.Any("error", err))
	}
}

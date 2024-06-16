package chunk_server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/cenkalti/backoff"

	"simple-s3-adventure/pkg/config"
	"simple-s3-adventure/pkg/logger"
)

const (
	uuidPattern       = "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"
	registrationDelay = 5 * time.Second
)

var uuidRegex = regexp.MustCompile(uuidPattern)

type ServerConfig struct {
	Port               string
	UploadDir          string
	FrontServerAddress string
	MaxUploadSize      int64
}

func NewServerConfig() *ServerConfig {
	cfg := &ServerConfig{
		Port:               config.GetEnvString("PORT", "12090"),
		UploadDir:          config.GetEnvString("UPLOAD_DIR", "tmp"),
		FrontServerAddress: config.GetEnvString("FRONT_SERVER_ADDRESS", "http://front-server:13090"),
		MaxUploadSize:      config.GetEnvInt64("MAX_UPLOAD_SIZE", 10<<20),
	}

	if err := validatePort(cfg.Port); err != nil {
		lg := logger.GetLogger()
		lg.Error("Invalid port number", slog.String("port", cfg.Port))
		os.Exit(1)
	}

	return cfg
}

// validatePort checks if the given port is valid.
func validatePort(port string) error {
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}
	return nil
}

func registerHandlers(mux *http.ServeMux, config *ServerConfig) {
	mux.HandleFunc("/put", func(w http.ResponseWriter, r *http.Request) {
		PutHandler(w, r, config)
	})
	mux.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		GetHandler(w, r, config)
	})
}

// StartServer starts the HTTP server on the given port.
func StartServer(config *ServerConfig) {
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
		lg.Info("Chunk server registered")
	})

	lg.Info("Starting chunk server", slog.String("port", config.Port))
	address := net.JoinHostPort("", config.Port)
	if err := http.ListenAndServe(address, mux); err != nil {
		lg.Error("Could not start server", slog.Any("error", err))
	}
}

package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"simple-s3-adventure/internal/front_server/front_service"
	"syscall"
	"time"

	"simple-s3-adventure/internal/front_server/registry_service"
	"simple-s3-adventure/pkg/logger"
)

const shutdownTimeout = 5 * time.Second

type FrontServer struct {
	service *front_service.FrontService
	server  *http.Server
}

func NewFrontServer() *FrontServer {
	return &FrontServer{
		service: front_service.NewFrontService(
			registry_service.NewChunkServerRegistry(),
			registry_service.NewChunkAllocationMap()),
	}
}

// StartServer starts the HTTP server on the given port.
func StartServer(ctx context.Context, port string) {
	server := NewFrontServer()
	lg := logger.GetLogger()

	// Setting up handlers
	http.HandleFunc("/register_chunk_server", server.RegisterChunkServerHandler)
	http.HandleFunc("/put", server.PutHandler)
	http.HandleFunc("/get", server.GetHandler)

	// Create the HTTP server
	server.server = &http.Server{
		Addr:    port,
		Handler: nil,
	}

	// Starting the server
	lg.Info("Starting front server", slog.String("port", port))
	go func() {
		if err := server.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			lg.Error("Could not start server: ", err)
		}
	}()

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done

	lg.Info("Shutting down front server")
	ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	if err := server.server.Shutdown(ctx); err != nil {
		lg.Error("Server shutdown failed:", err)
	} else {
		lg.Info("Server shutdown gracefully")
	}
}

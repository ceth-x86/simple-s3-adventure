package front_server

import (
	"container/list"
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"simple-s3-adventure/pkg/logger"
)

const shutdownTimeout = 5 * time.Second

type FrontServer struct {
	registry      *chunkServerRegistry
	allocationMap *chunkAllocationMap
	httpClient    *http.Client
	server        *http.Server
}

func NewFrontServer() *FrontServer {
	return &FrontServer{
		registry: &chunkServerRegistry{
			chunkServerAddresses: make(map[string]struct{}),
			chunkServers:         list.New(),
		},
		allocationMap: &chunkAllocationMap{chunks: make(map[string][]*chunkServer)},

		httpClient: &http.Client{},
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
		if err := server.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

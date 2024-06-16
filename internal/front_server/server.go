package front_server

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
)

type FrontServer struct {
	chunkServers map[string]struct{}
	mu           sync.RWMutex

	logger *slog.Logger
}

// StartServer starts the HTTP server on the given port.
func StartServer(ctx context.Context, logger *slog.Logger, port string) {
	server := &FrontServer{
		chunkServers: make(map[string]struct{}),
		logger:       logger,
	}
	// Setting up handlers
	http.HandleFunc("/register_chunk_server", server.RegisterChunkServerHandler)
	http.HandleFunc("/upload", server.UploadHandler)

	// Starting the server
	logger.Info("Starting front server", slog.String("port", port))
	if err := http.ListenAndServe(port, nil); err != nil {
		logger.Error("Could not start server: ", err)
	}
}

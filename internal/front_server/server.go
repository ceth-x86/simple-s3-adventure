package front_server

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
)

type FrontServer struct {
	// chunkServers is a set of chunk server addresses.
	chunkServers map[string]struct{}
	muServers    sync.RWMutex

	// fileParts is a map of file UUIDs to their parts.
	chunks   map[string][]string
	muChunks sync.RWMutex

	httpClient *http.Client
	logger     *slog.Logger
}

// StartServer starts the HTTP server on the given port.
func StartServer(ctx context.Context, logger *slog.Logger, port string) {
	server := &FrontServer{
		chunkServers: make(map[string]struct{}),
		chunks:       make(map[string][]string),
		httpClient:   &http.Client{},
		logger:       logger,
	}
	// Setting up handlers
	http.HandleFunc("/register_chunk_server", server.RegisterChunkServerHandler)
	http.HandleFunc("/put", server.PutHandler)
	http.HandleFunc("/get", server.GetHandler)

	// Starting the server
	logger.Info("Starting front server", slog.String("port", port))
	if err := http.ListenAndServe(port, nil); err != nil {
		logger.Error("Could not start server: ", err)
	}
}

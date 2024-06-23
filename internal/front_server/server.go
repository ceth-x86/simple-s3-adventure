package front_server

import (
	"container/list"
	"context"
	"log/slog"
	"net/http"
)

type FrontServer struct {
	registry      *chunkServerRegistry
	allocationMap *chunkAllocationMap
	httpClient    *http.Client
	logger        *slog.Logger
}

func NewFrontServer(logger *slog.Logger) *FrontServer {
	return &FrontServer{
		registry: &chunkServerRegistry{
			chunkServerAddresses: make(map[string]struct{}),
			chunkServers:         list.New(),
		},
		allocationMap: &chunkAllocationMap{chunks: make(map[string][]*chunkServer)},

		httpClient: &http.Client{},
		logger:     logger,
	}
}

// StartServer starts the HTTP server on the given port.
func StartServer(ctx context.Context, logger *slog.Logger, port string) {
	server := NewFrontServer(logger)

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

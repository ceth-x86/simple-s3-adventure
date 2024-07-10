package api

import (
	"log/slog"
	"net/http"

	chService "simple-s3-adventure/internal/chunk_server/service"
	"simple-s3-adventure/pkg/logger"
	uuid2 "simple-s3-adventure/pkg/uuid"
)

func GetHandler(w http.ResponseWriter, r *http.Request, config *chService.ServerConfig) {
	lg := logger.GetLogger()
	chunkService := chService.NewChunkService(config, lg)

	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	uuid := r.URL.Query().Get("uuid")
	if err := uuid2.Validate(uuid); err != nil {
		lg.Error("Incorrect UUID", slog.String("uuid", uuid))
		http.Error(w, "Incorrect UUID", http.StatusBadRequest)
		return
	}

	if err := chunkService.CopyFileToResponse(uuid, w); err != nil {
		lg.Error("Failed to copy file to response", slog.String("uuid", uuid), slog.Any("error", err))
		http.Error(w, "Failed to copy file", http.StatusInternalServerError)
		return
	}

	lg.Info("File sent", slog.String("uuid", uuid))
}

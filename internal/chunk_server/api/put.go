package api

import (
	"log/slog"
	"net/http"

	srv "simple-s3-adventure/internal/chunk_server/service"
	"simple-s3-adventure/pkg/logger"
	uuid2 "simple-s3-adventure/pkg/uuid"
)

func PutHandler(w http.ResponseWriter, r *http.Request, config *srv.ServerConfig) {
	lg := logger.GetLogger()
	chunkService := srv.NewChunkService(config, lg)

	if r.Method != http.MethodPut {
		lg.Error("Method not allowed", slog.String("method", r.Method))
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	uuid := r.FormValue("uuid")
	if err := uuid2.Validate(uuid); err != nil {
		http.Error(w, "Incorrect UUID", http.StatusBadRequest)
		return
	}

	if err := srv.CreateUploadDir(config.UploadDir); err != nil {
		http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
		return
	}

	// up to a total of 10MB bytes of the file are stored in memory,
	// with the remainder stored on disk in temporary files.
	err := r.ParseMultipartForm(config.MaxUploadSize)
	if err != nil {
		lg.Error("Failed to parse multipart form", slog.Any("error", err))
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		lg.Error("Failed to get file from form", slog.Any("error", err))
		http.Error(w, "failed to get file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if err := chunkService.SaveUploadedFile(file, uuid); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

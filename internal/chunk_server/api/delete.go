package api

import (
	"log/slog"
	"net/http"

	srv "simple-s3-adventure/internal/chunk_server/service"
	"simple-s3-adventure/pkg/logger"
	uuid2 "simple-s3-adventure/pkg/uuid"
)

// DeleteHandler deletes the file with the given UUID from the server.
func DeleteHandler(w http.ResponseWriter, r *http.Request, config *srv.ServerConfig) {
	lg := logger.GetLogger()

	if r.Method != http.MethodDelete {
		lg.Error("Method not allowed", slog.String("method", r.Method))
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	uuid := r.FormValue("uuid")
	if err := uuid2.Validate(uuid); err != nil {
		lg.Error("Incorrect UUID", slog.String("uuid", uuid))
		http.Error(w, "Incorrect UUID", http.StatusBadRequest)
		return
	}

	err := srv.DeleteFile(config.UploadDir, uuid)
	if err != nil {
		if err.Error() == "file not found" {
			lg.Error("File not found", slog.String("uuid", uuid))
			http.Error(w, "File not found", http.StatusNotFound)
		} else {
			lg.Error("Failed to delete file on server", slog.String("uuid", uuid), slog.Any("error", err))
			http.Error(w, "Failed to delete file on server", http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
}

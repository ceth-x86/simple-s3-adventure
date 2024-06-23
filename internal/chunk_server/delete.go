package chunk_server

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"simple-s3-adventure/pkg/logger"
)

// DeleteHandler deletes the file with the given UUID from the server.
func DeleteHandler(w http.ResponseWriter, r *http.Request, config *ServerConfig) {
	lg := logger.GetLogger()

	if r.Method != http.MethodDelete {
		lg.Error("Method not allowed", slog.String("method", r.Method))
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	uuid := r.FormValue("uuid")
	if !uuidRegex.MatchString(uuid) {
		lg.Error("Incorrect UUID", slog.String("uuid", uuid))
		http.Error(w, "Incorrect UUID", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(config.UploadDir, uuid)
	err := os.Remove(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			lg.Warn("File not found", slog.String("filePath", filePath))
			http.Error(w, "File not found", http.StatusNotFound)
		} else {
			lg.Error("Failed to delete file on server", slog.Any("error", err))
			http.Error(w, "Failed to delete file on server", http.StatusInternalServerError)
		}
		return
	}

	lg.Info("File deleted", slog.String("uuid", uuid))
	w.WriteHeader(http.StatusOK)
}

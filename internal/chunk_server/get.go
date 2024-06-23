package chunk_server

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"simple-s3-adventure/pkg/logger"
)

func GetHandler(w http.ResponseWriter, r *http.Request, config *ServerConfig) {
	lg := logger.GetLogger()
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	uuid := r.URL.Query().Get("uuid")
	if !uuidRegex.MatchString(uuid) {
		http.Error(w, "Incorrect UUID", http.StatusBadRequest)
		return
	}

	path := filepath.Join(config.UploadDir, uuid)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	f, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		http.Error(w, "Failed to open file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if _, err := io.Copy(w, f); err != nil {
		http.Error(w, "Failed to copy file", http.StatusInternalServerError)
		return
	}

	lg.Info("File sent", slog.String("uuid", uuid))
}

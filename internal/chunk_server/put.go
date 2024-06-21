package chunk_server

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"simple-s3-adventure/pkg/logger"
)

func PutHandler(w http.ResponseWriter, r *http.Request, config *ServerConfig) {
	lg := logger.GetLogger()
	if r.Method != http.MethodPut {
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

	if err := os.MkdirAll(config.UploadDir, os.ModePerm); err != nil {
		lg.Error("Failed to create upload directory", slog.Any("error", err))
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

	// Create a file on the server to save the uploaded file
	filePath := filepath.Join(config.UploadDir, uuid)
	dst, err := os.Create(filePath)
	if err != nil {
		lg.Error("Failed to create file on server", slog.Any("error", err))
		http.Error(w, "failed to create file on server", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		lg.Error("Failed to save uploaded file", slog.Any("error", err))
		http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
		return
	}

	lg.Info("File uploaded", slog.String("file_id", uuid))
	w.WriteHeader(http.StatusOK)
}

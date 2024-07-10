package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"simple-s3-adventure/pkg/config"
	"simple-s3-adventure/pkg/logger"
)

const (
	defaultNumParts      = 6
	defaultMaxUploadSize = 10 << 20 // 10 MB
)

var (
	numParts      = config.GetEnvInt("NUM_PARTS", defaultNumParts)
	maxUploadSize = config.GetEnvInt64("MAX_UPLOAD_SIZE", defaultMaxUploadSize)
)

type putResponse struct {
	UUID string `json:"uuid"`
}

func httpError(res http.ResponseWriter, message string, statusCode int, err error) {
	if err != nil {
		logger.GetLogger().Error(message, slog.Any("error", err))
	}
	http.Error(res, message, statusCode)
}

func (f *FrontServer) PutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	fileUUID, err := f.service.UploadFile(r, maxUploadSize, numParts)
	if err != nil {
		httpError(w, "Failed to upload file", http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	result := putResponse{UUID: fileUUID}
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		httpError(w, "Failed to encode response", http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

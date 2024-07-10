package api

import (
	"net/http"
	uuid2 "simple-s3-adventure/pkg/uuid"
	"strconv"
)

func (f *FrontServer) GetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	uuid := r.URL.Query().Get("uuid")
	if err := uuid2.Validate(uuid); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	size, err := f.service.CopyChunks(uuid, w)
	if err != nil {
		http.Error(w, "Error copying chunks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment")
	w.WriteHeader(http.StatusOK)
}

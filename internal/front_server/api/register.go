package api

import (
	"log/slog"
	"net/http"

	"simple-s3-adventure/pkg/logger"
)

func (f *FrontServer) RegisterChunkServerHandler(w http.ResponseWriter, r *http.Request) {
	lg := logger.GetLogger()

	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		lg.Error("Method not allowed", slog.String("method", r.Method))
		return
	}

	serverURL := r.FormValue("url")
	err := f.service.RegisterChunkServer(serverURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Chunk server registered successfully"))
}

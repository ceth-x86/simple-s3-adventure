package front_server

import (
	"log/slog"
	"net/http"
	"net/url"

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
	if serverURL == "" {
		httpError(w, "URL not provided", http.StatusBadRequest, nil, f)
		return
	}

	parsedURL, err := url.ParseRequestURI(serverURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		lg.Error("Invalid URL", slog.String("url", serverURL))
		return
	}

	lg.Info("Registering chunk server", slog.String("url", r.FormValue("url")))
	err = f.registry.addChunkServer(serverURL)
	if err != nil {
		lg.Error(err.Error())
		http.Error(w, err.Error(), http.StatusConflict)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Chunk server registered successfully"))
}

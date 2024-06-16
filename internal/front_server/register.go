package front_server

import (
	"log/slog"
	"net/http"
	"net/url"
)

func (f *FrontServer) RegisterChunkServerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		f.logger.Error("Method not allowed", slog.String("method", r.Method))
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
		f.logger.Error("Invalid URL", slog.String("url", serverURL))
		return
	}

	f.logger.Info("Registering chunk server", slog.String("url", r.FormValue("url")))

	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.chunkServers[serverURL]; exists {
		http.Error(w, "Server already registered", http.StatusConflict)
		f.logger.Error("Server already registered", slog.String("url", serverURL))
		return
	}
	f.chunkServers[serverURL] = struct{}{}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Chunk server registered successfully"))
}

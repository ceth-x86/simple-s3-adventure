package chunk_server

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"simple-s3-adventure/pkg/config"
	"simple-s3-adventure/pkg/logger"
)

const uuidPattern = "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"

var uuidRegex = regexp.MustCompile(uuidPattern)

type ServerConfig struct {
	Port          string
	UploadDir     string
	MaxUploadSize int64
}

func NewServerConfig() *ServerConfig {
	config := &ServerConfig{
		Port:          config.GetEnvString("PORT", "12090"),
		UploadDir:     config.GetEnvString("UPLOAD_DIR", "tmp"),
		MaxUploadSize: config.GetEnvInt64("MAX_UPLOAD_SIZE", 10<<20),
	}

	if _, err := strconv.Atoi(config.Port); err != nil {
		lg := logger.GetLogger()
		lg.Error("Invalid port number", slog.String("port", config.Port))
		os.Exit(1)
	}

	return config
}

func registerHandlers(mux *http.ServeMux, config *ServerConfig) {
	mux.HandleFunc("/put", func(w http.ResponseWriter, r *http.Request) {
		PutHandler(w, r, config)
	})
	mux.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		GetHandler(w, r, config)
	})
}

// StartServer starts the HTTP server on the given port.
func StartServer(config *ServerConfig) {
	mux := http.NewServeMux()
	registerHandlers(mux, config)

	lg := logger.GetLogger()
	lg.Info("Starting chunk server", slog.String("port", config.Port))
	address := net.JoinHostPort("", config.Port)
	if err := http.ListenAndServe(address, mux); err != nil {
		lg.Error("Could not start server: ", slog.Any("error", err))
	}
}

package service

import (
	"fmt"
	"log/slog"
	"os"
	"simple-s3-adventure/pkg/config"
	"simple-s3-adventure/pkg/logger"
	"strconv"
)

type ServerConfig struct {
	Port               string
	UploadDir          string
	FrontServerAddress string
	MaxUploadSize      int64
}

func NewServerConfig() *ServerConfig {
	cfg := &ServerConfig{
		Port:               config.GetEnvString("PORT", "12090"),
		UploadDir:          config.GetEnvString("UPLOAD_DIR", "tmp"),
		FrontServerAddress: config.GetEnvString("FRONT_SERVER_ADDRESS", "http://front-server:13090"),
		MaxUploadSize:      config.GetEnvInt64("MAX_UPLOAD_SIZE", 10<<20),
	}

	if err := validatePort(cfg.Port); err != nil {
		lg := logger.GetLogger()
		lg.Error("Invalid port number", slog.String("port", cfg.Port))
		os.Exit(1)
	}

	return cfg
}

// validatePort checks if the given port is valid.
func validatePort(port string) error {
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}
	return nil
}

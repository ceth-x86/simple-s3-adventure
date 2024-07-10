package service

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

type ChunkService struct {
	Config *ServerConfig
	Logger *slog.Logger
}

func NewChunkService(config *ServerConfig, logger *slog.Logger) *ChunkService {
	return &ChunkService{
		Config: config,
		Logger: logger,
	}
}

func (cs *ChunkService) SaveUploadedFile(r io.Reader, uuid string) error {
	filePath := filepath.Join(cs.Config.UploadDir, uuid)
	dst, err := os.Create(filePath)
	if err != nil {
		cs.Logger.Error("Failed to create file on server", slog.Any("error", err))
		return fmt.Errorf("failed to create file on server")
	}
	defer dst.Close()

	if _, err := io.Copy(dst, r); err != nil {
		cs.Logger.Error("Failed to save uploaded file", slog.Any("error", err))
		return fmt.Errorf("failed to save uploaded file")
	}

	cs.Logger.Info("File uploaded", slog.String("uuid", uuid))
	return nil
}

func (cs *ChunkService) CopyFileToResponse(uuid string, w http.ResponseWriter) error {
	exists, err := fileExists(cs.Config.UploadDir, uuid)
	if err != nil {
		cs.Logger.Error("Failed to check file existence", slog.Any("error", err))
		return fmt.Errorf("failed to check file existence")
	}
	if !exists {
		return fmt.Errorf("file not found")
	}

	f, err := openFile(cs.Config.UploadDir, uuid)
	if err != nil {
		cs.Logger.Error("Failed to open file", slog.Any("error", err))
		return err
	}
	defer f.Close()

	if _, err := io.Copy(w, f); err != nil {
		cs.Logger.Error("Failed to copy file", slog.Any("error", err))
		return fmt.Errorf("failed to copy file")
	}
	return nil
}

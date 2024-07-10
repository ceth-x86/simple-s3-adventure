package front_service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"simple-s3-adventure/internal/front_server/chunker"
	"simple-s3-adventure/internal/front_server/upload_service"
	"simple-s3-adventure/pkg/logger"

	"github.com/google/uuid"
)

func (s *FrontService) UploadFile(r *http.Request, maxUploadSize int64, numParts int) (string, error) {
	fileUUID := uuid.New().String()

	err := r.ParseMultipartForm(maxUploadSize)
	if err != nil {
		return "", fmt.Errorf("failed to parse multipart form: %w", err)
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		return "", fmt.Errorf("failed to get file from form: %w", err)
	}
	defer file.Close()

	lg := logger.GetLogger()
	lg.Info("File uploading", slog.String("file_id", fileUUID), slog.Int64("file_size", header.Size))

	offsets := chunker.ChunkOffsets(header.Size, numParts)
	servers := s.registry.SelectUnderloadedChunkServers(numParts)
	if len(servers) != numParts {
		return "", fmt.Errorf("not enough chunk servers available")
	}

	uploadService := upload_service.NewUploadService(s.httpClient, s.registry, s.allocationMap)

	chunks := upload_service.CreateChunks(header.Size, offsets, servers)

	ctx := context.Background()
	if err := uploadService.ProcessFileChunks(ctx, file, fileUUID, chunks); err != nil {
		// We tried to write the file to the server, but we couldn’t.
		// The best we can do now is clean up after ourselves and return an error.

		// Some servers that failed to write the file will respond with an error when attempting to delete it.
		// This is not critical; we can return to this optimization later.
		if delErr := uploadService.DeleteFileChunks(ctx, fileUUID, chunks); delErr != nil {
			lg.Warn("Failed to delete file chunks", slog.String("file_id", fileUUID), slog.Any("error", delErr))
		}

		return "", fmt.Errorf("failed to process chunk: %w", err)
	}

	s.allocationMap.AddChunk(fileUUID, servers)

	// Update the size of the chunk servers
	incSizes := make([]int64, len(servers))
	for i := range servers {
		incSizes[i] = upload_service.CalculateChunkSize(header.Size, offsets, i)
	}
	s.registry.AdjustSizes(servers, incSizes, header.Size)

	return fileUUID, nil
}

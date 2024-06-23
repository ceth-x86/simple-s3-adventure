package front_server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"time"

	"simple-s3-adventure/pkg/config"
	"simple-s3-adventure/pkg/logger"

	"github.com/cenkalti/backoff"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
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

func httpError(res http.ResponseWriter, message string, statusCode int, err error, f *FrontServer) {
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

	fileUUID := uuid.New().String()

	err := r.ParseMultipartForm(maxUploadSize)
	if err != nil {
		httpError(w, "Failed to parse multipart form", http.StatusInternalServerError, err, f)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpError(w, "Failed to get file from form", http.StatusInternalServerError, err, f)
		return
	}
	defer file.Close()

	lg := logger.GetLogger()
	lg.Info("File uploading", slog.String("file_id", fileUUID), slog.Int64("file_size", header.Size))

	offsets := chunkOffsets(header.Size, numParts)
	servers := f.registry.selectUnderloadedChunkServers(numParts)
	if len(servers) != numParts {
		httpError(w, "Not enough chunk servers available", http.StatusInternalServerError, nil, f)
		return
	}

	ctx := context.Background()
	if err := f.processFileChunks(ctx, file, fileUUID, header.Size, offsets, servers); err != nil {
		// We tried to write the file to the server, but we couldn’t.
		// The best we can do now is clean up after ourselves and return an error.

		// Some servers that failed to write the file will respond with an error when attempting to delete it.
		// This is not critical; we can return to this optimization later.
		if delErr := f.deleteFileChunks(ctx, fileUUID, servers); delErr != nil {
			lg.Warn("Failed to delete file chunks", slog.String("file_id", fileUUID), slog.Any("error", delErr))
		}

		httpError(w, "Failed to process chunk", http.StatusInternalServerError, err, f)
		return
	}

	f.allocationMap.addChunk(fileUUID, servers)

	// Update the size of the chunk servers
	incSizes := make([]int64, len(servers))
	for i, _ := range servers {
		incSizes[i] = calculateChunkSize(header.Size, offsets, i)

	}
	f.registry.adjustSizes(servers, incSizes, header.Size)

	w.Header().Set("Content-Type", "text/json")
	result := putResponse{UUID: fileUUID}
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		httpError(w, "Failed to encode response", http.StatusInternalServerError, err, f)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (f *FrontServer) processFileChunks(ctx context.Context, file multipart.File, uuid string, fileSize int64, offsets []int64, servers []*chunkServer) error {
	g, ctx := errgroup.WithContext(ctx)

	for i := 0; i < len(offsets); i++ {
		i := i
		g.Go(func() error {
			return f.processChunk(ctx, file, uuid, fileSize, offsets, servers, i)
		})
	}

	return g.Wait()
}

// processChunk processes a single chunk of the uploaded file.
func (f *FrontServer) processChunk(ctx context.Context, file multipart.File, uuid string, fileSize int64, offsets []int64, servers []*chunkServer, i int) error {
	startOffset := offsets[i]
	chunkSize := calculateChunkSize(fileSize, offsets, i)

	lg := logger.GetLogger()
	lg.Info("Processing chunk", slog.String("uuid", uuid), slog.Int("chunk", i), slog.String("server", servers[i].address), slog.Int64("start_offset", startOffset), slog.Int64("chunk_size", chunkSize))

	sr := io.NewSectionReader(file, startOffset, chunkSize)
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	if err := writer.WriteField("uuid", uuid); err != nil {
		return fmt.Errorf("failed to add UUID field: %w", err)
	}

	part, err := writer.CreateFormFile("file", "file.txt")
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, sr); err != nil {
		return fmt.Errorf("failed to copy chunk to part: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	req, err := http.NewRequest("PUT", servers[i].address+"/put", &requestBody)
	if err != nil {
		return fmt.Errorf("failed to create PUT request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var attempt int
	var resp *http.Response

	// Create a backoff policy with a maximum of 5 retries
	bo := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5)
	bo = backoff.WithContext(bo, ctx)

	if err := backoff.Retry(func() error {
		attempt++
		resp, err = f.httpClient.Do(req.WithContext(ctx))
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			lg.Error("Failed to send PUT request", slog.Int("attempt", attempt), slog.String("error", err.Error()))
			return fmt.Errorf("failed to send PUT request: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("received non-OK HTTP status: %d", resp.StatusCode)
		}
		return nil
	}, bo); err != nil {
		if errors.Is(err, context.Canceled) {
			lg.Error("Uploading cancelled")
		} else {
			lg.Error("Failed to send request to chunk server", slog.String("error", err.Error()))
		}
		return err
	}
	return nil
}

func (f *FrontServer) deleteFileChunks(ctx context.Context, uuid string, servers []*chunkServer) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, server := range servers {
		server := server
		g.Go(func() error {
			return f.deleteChunk(ctx, uuid, server)
		})
	}

	return g.Wait()
}

func (f *FrontServer) deleteChunk(ctx context.Context, uuid string, server *chunkServer) error {
	req, err := http.NewRequest("DELETE", server.address+"/delete?uuid="+uuid, nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var attempt int
	var resp *http.Response
	lg := logger.GetLogger()

	// Create a backoff policy with a maximum of 5 retries
	bo := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5)
	bo = backoff.WithContext(bo, ctx)

	if err := backoff.Retry(func() error {
		attempt++
		resp, err = f.httpClient.Do(req.WithContext(ctx))
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			lg.Error("Failed to send DELETE request", slog.Int("attempt", attempt), slog.String("error", err.Error()))
			return fmt.Errorf("failed to send DELETE request: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("received non-OK HTTP status: %d", resp.StatusCode)
		}
		return nil
	}, bo); err != nil {
		if errors.Is(err, context.Canceled) {
			lg.Error("Deletion cancelled")
		} else {
			lg.Error("Failed to send request to chunk server", slog.String("error", err.Error()))
		}
		return err
	}
	return nil

}

// calculateChunkSize calculates the size of a chunk based on its offsets.
func calculateChunkSize(fileSize int64, offsets []int64, i int) int64 {
	if i != len(offsets)-1 {
		return offsets[i+1] - offsets[i]
	}
	return fileSize - offsets[i]
}

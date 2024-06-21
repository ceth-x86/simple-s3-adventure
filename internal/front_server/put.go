package front_server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"simple-s3-adventure/pkg/config"
	"time"

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

func httpError(res http.ResponseWriter, message string, statusCode int, err error, f *FrontServer) {
	if err != nil {
		f.logger.Error(message, slog.Any("error", err))
	}
	http.Error(res, message, statusCode)
}

func (f *FrontServer) PutHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPut {
		http.Error(res, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	newUUID := uuid.New()

	err := req.ParseMultipartForm(maxUploadSize)
	if err != nil {
		httpError(res, "Failed to parse multipart form", http.StatusInternalServerError, err, f)
		return
	}

	file, header, err := req.FormFile("file")
	if err != nil {
		httpError(res, "Failed to get file from form", http.StatusInternalServerError, err, f)
		return
	}
	defer file.Close()

	f.logger.Info("File uploading", slog.String("file_id", newUUID.String()), slog.Int64("file_size", header.Size))

	offsets := chunkOffsets(header.Size, numParts)
	servers := f.selectFirstNChunkServers(numParts)
	if len(servers) != numParts {
		httpError(res, "Not enough chunk servers available", http.StatusInternalServerError, nil, f)
		return
	}

	if err := f.processFileChunks(context.Background(), file, newUUID.String(), header.Size, offsets, servers); err != nil {
		httpError(res, "Failed to process chunk", http.StatusInternalServerError, err, f)
		// TODO: we need to delete the uploaded chunks
		return
	}

	res.WriteHeader(http.StatusOK)
}

func (f *FrontServer) processFileChunks(ctx context.Context, file multipart.File, uuid string, fileSize int64, offsets []int64, servers []string) error {
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
func (f *FrontServer) processChunk(ctx context.Context, file multipart.File, uuid string, fileSize int64, offsets []int64, servers []string, i int) error {
	startOffset := offsets[i]
	chunkSize := calculateChunkSize(fileSize, offsets, i)

	f.logger.Info("Processing chunk", slog.String("uuid", uuid), slog.Int("chunk", i), slog.String("server", servers[i]), slog.Int64("start_offset", startOffset), slog.Int64("chunk_size", chunkSize))

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

	req, err := http.NewRequest("PUT", servers[i]+"/put", &requestBody)
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
			f.logger.Info("Failed to send PUT request", slog.Int("attempt", attempt), slog.String("error", err.Error()))
			return fmt.Errorf("failed to send PUT request: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("received non-OK HTTP status: %d", resp.StatusCode)
		}
		return nil
	}, bo); err != nil {
		if errors.Is(err, context.Canceled) {
			f.logger.Info("Registration cancelled")
		} else {
			f.logger.Info("Failed to send request to chunk server", slog.String("error", err.Error()))
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

package upload_service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"time"

	"simple-s3-adventure/internal/front_server/registry_service"
	"simple-s3-adventure/pkg/logger"

	"github.com/cenkalti/backoff"
	"golang.org/x/sync/errgroup"
)

type UploadService struct {
	httpClient    *http.Client
	registry      *registry_service.ChunkServerRegistry
	allocationMap *registry_service.ChunkAllocationMap
}

func NewUploadService(httpClient *http.Client, registry *registry_service.ChunkServerRegistry, allocationMap *registry_service.ChunkAllocationMap) *UploadService {
	return &UploadService{
		httpClient:    httpClient,
		registry:      registry,
		allocationMap: allocationMap,
	}
}

func (u *UploadService) ProcessFileChunks(ctx context.Context, file multipart.File, uuid string, chunks []*Chunk) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, chunk := range chunks {
		chunk := chunk
		g.Go(func() error {
			return u.processChunk(ctx, file, uuid, chunk)
		})
	}

	return g.Wait()
}

func (u *UploadService) processChunk(ctx context.Context, file multipart.File, uuid string, chunk *Chunk) error {
	lg := logger.GetLogger()
	lg.Info("Processing chunk", slog.String("uuid", uuid), slog.Int("chunk", chunk.Index), slog.String("server", chunk.Server.Address), slog.Int64("start_offset", chunk.StartOffset), slog.Int64("chunk_size", chunk.Size))

	sr := io.NewSectionReader(file, chunk.StartOffset, chunk.Size)
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

	req, err := http.NewRequest("PUT", chunk.Server.Address+"/put", &requestBody)
	if err != nil {
		return fmt.Errorf("failed to create PUT request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var attempt int
	var resp *http.Response

	bo := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5)
	bo = backoff.WithContext(bo, ctx)

	if err := backoff.Retry(func() error {
		attempt++
		resp, err = u.httpClient.Do(req.WithContext(ctx))
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

func (u *UploadService) DeleteFileChunks(ctx context.Context, uuid string, chunks []*Chunk) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, chunk := range chunks {
		chunk := chunk
		g.Go(func() error {
			return u.deleteChunk(ctx, uuid, chunk.Server)
		})
	}

	return g.Wait()
}

func (u *UploadService) deleteChunk(ctx context.Context, uuid string, server *registry_service.ChunkServer) error {
	req, err := http.NewRequest("DELETE", server.Address+"/delete?uuid="+uuid, nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var attempt int
	var resp *http.Response
	lg := logger.GetLogger()

	bo := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5)
	bo = backoff.WithContext(bo, ctx)

	if err := backoff.Retry(func() error {
		attempt++
		resp, err = u.httpClient.Do(req.WithContext(ctx))
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

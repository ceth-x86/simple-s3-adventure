package api

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"simple-s3-adventure/pkg/logger"
	"time"
)

const requestTimeout = 30 * time.Second

func register(frontServerAddress string, chunkServerPort string) error {
	hostname, err := os.Hostname()
	if err != nil {
		return logAndReturnError(fmt.Errorf("failed to get hostname: %w", err))
	}

	url := fmt.Sprintf("http://%s:%s", hostname, chunkServerPort)

	lg := logger.GetLogger()
	lg.Info("Registering chunk server", slog.String("front_server", frontServerAddress), slog.String("url", url))

	requestBody, contentType, err := createRequestBody(url)
	if err != nil {
		return logAndReturnError(err)
	}

	req, err := http.NewRequest("PUT", frontServerAddress+"/register_chunk_server", requestBody)
	if err != nil {
		return logAndReturnError(fmt.Errorf("failed to create PUT request: %w", err))
	}

	req.Header.Set("Content-Type", contentType)

	client := &http.Client{}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return logAndReturnError(fmt.Errorf("failed to send PUT request: %w", err))
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			lg.Error("Failed to close response body", slog.Any("error", closeErr))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK HTTP status: %d", resp.StatusCode)
	}
	return nil
}

func createRequestBody(url string) (*bytes.Buffer, string, error) {
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	if err := writer.WriteField("url", url); err != nil {
		return nil, "", fmt.Errorf("failed to add URL field: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close writer: %w", err)
	}
	return &requestBody, writer.FormDataContentType(), nil
}

func logAndReturnError(err error) error {
	logger.GetLogger().Error(err.Error())
	return err
}

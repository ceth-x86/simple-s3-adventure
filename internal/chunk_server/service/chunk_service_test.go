package service

import (
	"bytes"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveUploadedFile(t *testing.T) {
	tempDir := t.TempDir()
	config := &ServerConfig{UploadDir: tempDir}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cs := NewChunkService(config, logger)

	fileContent := []byte("test content")
	fileReader := bytes.NewReader(fileContent)
	uuid := "test-uuid"

	err := cs.SaveUploadedFile(fileReader, uuid)
	require.NoError(t, err)

	savedFilePath := filepath.Join(tempDir, uuid)
	savedFileContent, err := os.ReadFile(savedFilePath)
	require.NoError(t, err)
	assert.Equal(t, fileContent, savedFileContent)
}

func TestCopyFileToResponse(t *testing.T) {
	tempDir := t.TempDir()
	config := &ServerConfig{UploadDir: tempDir}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cs := NewChunkService(config, logger)

	fileContent := []byte("test content")
	filePath := filepath.Join(tempDir, "test-uuid")
	require.NoError(t, os.WriteFile(filePath, fileContent, 0644))

	w := &fakeResponseWriter{
		header: http.Header{},
	}
	err := cs.CopyFileToResponse("test-uuid", w)
	require.NoError(t, err)
	assert.Equal(t, fileContent, w.body.Bytes())
}

func TestCopyFileToResponse_FileNotFound(t *testing.T) {
	config := &ServerConfig{UploadDir: t.TempDir()}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cs := NewChunkService(config, logger)

	w := &fakeResponseWriter{
		header: http.Header{},
	}

	err := cs.CopyFileToResponse("non-existent-uuid", w)
	require.Error(t, err)
	assert.Equal(t, "file not found", err.Error())
}

func TestCopyFileToResponse_ErrorOpeningFile(t *testing.T) {
	config := &ServerConfig{UploadDir: "/invalid-dir"}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cs := NewChunkService(config, logger)

	w := &fakeResponseWriter{
		header: http.Header{},
	}

	err := cs.CopyFileToResponse("test-uuid", w)
	require.Error(t, err)
}

// fakeResponseWriter is a simple fake implementation of http.ResponseWriter for testing
type fakeResponseWriter struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func (f *fakeResponseWriter) Header() http.Header {
	return f.header
}

func (f *fakeResponseWriter) Write(data []byte) (int, error) {
	return f.body.Write(data)
}

func (f *fakeResponseWriter) WriteHeader(statusCode int) {
	f.status = statusCode
}

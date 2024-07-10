package download_service

import (
	"io"
	"log/slog"
	"net/http"
	"simple-s3-adventure/internal/front_server/registry_service"
	"simple-s3-adventure/pkg/logger"
	"sync"
)

type DownloadService struct {
	uuid         string
	chunkServers []*registry_service.ChunkServer
	readers      []*io.PipeReader
	writers      []*io.PipeWriter
	multErrs     []error
	mu           sync.Mutex
	wg           sync.WaitGroup
	logger       *slog.Logger
}

func NewDownloadService(uuid string, chunkServers []*registry_service.ChunkServer) *DownloadService {
	readers, writers := createPipes(len(chunkServers))
	return &DownloadService{
		uuid:         uuid,
		chunkServers: chunkServers,
		readers:      readers,
		writers:      writers,
		multErrs:     []error{},
		logger:       logger.GetLogger(),
	}
}

func (ccm *DownloadService) CopyChunks(w io.Writer) (int64, error) {
	for i, server := range ccm.chunkServers {
		ccm.wg.Add(1)
		go ccm.fetchChunk(i, server.Address)
	}

	ccm.closeWritersAfterCompletion()

	if len(ccm.multErrs) != 0 {
		ccm.logger.Error("Error fetching chunks", slog.Any("errors", ccm.multErrs))
		return 0, ccm.multErrs[0]
	}

	return ccm.copyChunksToWriter(w)
}

func (ccm *DownloadService) fetchChunk(i int, server string) {
	defer ccm.wg.Done()
	defer ccm.writers[i].Close()

	resp, err := http.Get(server + "/get?uuid=" + ccm.uuid)
	if err != nil {
		ccm.writers[i].CloseWithError(err)
		ccm.mu.Lock()
		ccm.multErrs = append(ccm.multErrs, err)
		ccm.mu.Unlock()
		return
	}
	defer resp.Body.Close()

	if _, err := io.Copy(ccm.writers[i], resp.Body); err != nil {
		ccm.writers[i].CloseWithError(err)
		ccm.mu.Lock()
		ccm.multErrs = append(ccm.multErrs, err)
		ccm.mu.Unlock()
	}
}

func (ccm *DownloadService) closeWritersAfterCompletion() {
	go func() {
		ccm.wg.Wait()
		for _, writer := range ccm.writers {
			writer.Close()
		}
	}()
}

func (ccm *DownloadService) copyChunksToWriter(w io.Writer) (int64, error) {
	var size int64
	for _, reader := range ccm.readers {
		n, err := io.Copy(w, reader)
		if err != nil {
			ccm.logger.Error("Error copying chunks", slog.Any("error", err))
			return size, err
		}
		size += n
	}
	return size, nil
}

func createPipes(n int) ([]*io.PipeReader, []*io.PipeWriter) {
	readers := make([]*io.PipeReader, n)
	writers := make([]*io.PipeWriter, n)
	for i := 0; i < n; i++ {
		readers[i], writers[i] = io.Pipe()
	}
	return readers, writers
}

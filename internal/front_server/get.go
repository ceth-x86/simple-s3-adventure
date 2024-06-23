package front_server

import (
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"sync"
)

const (
	uuidPattern = "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"
)

var uuidRegex = regexp.MustCompile(uuidPattern)

func (f *FrontServer) GetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	uuid := r.URL.Query().Get("uuid")
	if uuid == "" || !uuidRegex.MatchString(uuid) {
		http.Error(w, "Incorrect UUID", http.StatusBadRequest)
		return
	}

	chunkServers := f.allocationMap.getChunkServers(uuid)

	readers := make([]*io.PipeReader, len(chunkServers))
	writers := make([]*io.PipeWriter, len(chunkServers))
	multErrs := make([]error, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := range chunkServers {
		readers[i], writers[i] = io.Pipe()
	}

	for i, server := range chunkServers {
		wg.Add(1)
		go func(i int, server string) {
			defer wg.Done()
			defer writers[i].Close()

			resp, err := http.Get(server + "/get?uuid=" + uuid)
			if err != nil {
				writers[i].CloseWithError(err)
				mu.Lock()
				multErrs = append(multErrs, err)
				mu.Unlock()
				return
			}
			defer resp.Body.Close()

			_, err = io.Copy(writers[i], resp.Body)
			if err != nil {
				writers[i].CloseWithError(err)
				mu.Lock()
				multErrs = append(multErrs, err)
				mu.Unlock()
			}
		}(i, server.address)
	}

	go func() {
		wg.Wait()
		for _, writer := range writers {
			writer.Close()
		}
	}()

	size := int64(0)
	for _, reader := range readers {
		n, err := io.Copy(w, reader)
		if err != nil {
			f.logger.Error("Error copying chunks", slog.Any("error", err))
			http.Error(w, "Error copying chunks", http.StatusInternalServerError)
			return
		}
		size += n
	}

	if len(multErrs) != 0 {
		f.logger.Error("Error fetching chunks", slog.Any("errors", multErrs))
		http.Error(w, "Error fetching chunks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.WriteHeader(http.StatusOK)
}

package front_service

import (
	"io"
	"simple-s3-adventure/internal/front_server/download_service"
)

func (s *FrontService) CopyChunks(uuid string, w io.Writer) (int64, error) {
	chunkServers := s.allocationMap.GetChunkServers(uuid)
	srv := download_service.NewDownloadService(uuid, chunkServers)
	return srv.CopyChunks(w)
}

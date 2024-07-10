package upload_service

import "simple-s3-adventure/internal/front_server/registry_service"

type Chunk struct {
	Index       int
	StartOffset int64
	Size        int64
	Server      *registry_service.ChunkServer
}

// CreateChunks creates chunks of the file to be uploaded.
func CreateChunks(fileSize int64, offsets []int64, servers []*registry_service.ChunkServer) []*Chunk {
	chunks := make([]*Chunk, len(offsets))
	for i := range offsets {
		chunks[i] = &Chunk{
			Index:       i,
			StartOffset: offsets[i],
			Size:        CalculateChunkSize(fileSize, offsets, i),
			Server:      servers[i],
		}
	}
	return chunks
}

// CalculateChunkSize calculates the size of a chunk based on its offsets.
func CalculateChunkSize(fileSize int64, offsets []int64, i int) int64 {
	if i != len(offsets)-1 {
		return offsets[i+1] - offsets[i]
	}
	return fileSize - offsets[i]
}

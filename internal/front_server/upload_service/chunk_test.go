package upload_service

import (
	"testing"

	"simple-s3-adventure/internal/front_server/registry_service"

	"github.com/stretchr/testify/assert"
)

func TestCreateChunks(t *testing.T) {
	fileSize := int64(100)
	offsets := []int64{0, 25, 50, 75}
	servers := []*registry_service.ChunkServer{
		{Address: "server1"},
		{Address: "server2"},
		{Address: "server3"},
		{Address: "server4"},
	}

	expectedChunks := []*Chunk{
		{Index: 0, StartOffset: 0, Size: 25, Server: servers[0]},
		{Index: 1, StartOffset: 25, Size: 25, Server: servers[1]},
		{Index: 2, StartOffset: 50, Size: 25, Server: servers[2]},
		{Index: 3, StartOffset: 75, Size: 25, Server: servers[3]},
	}

	chunks := CreateChunks(fileSize, offsets, servers)
	assert.Equal(t, expectedChunks, chunks)
}

func TestCalculateChunkSize(t *testing.T) {
	fileSize := int64(100)
	offsets := []int64{0, 25, 50, 75}

	tests := []struct {
		index    int
		expected int64
	}{
		{index: 0, expected: 25},
		{index: 1, expected: 25},
		{index: 2, expected: 25},
		{index: 3, expected: 25},
	}

	for _, tt := range tests {
		size := CalculateChunkSize(fileSize, offsets, tt.index)
		assert.Equal(t, tt.expected, size)
	}
}

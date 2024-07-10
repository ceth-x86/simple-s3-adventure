package chunker

import (
	"reflect"
	"testing"
)

// Тестовая функция для ChunkSize
func TestChunkSize(t *testing.T) {
	tests := []struct {
		fileSize int64
		numParts int
		expected int64
	}{
		{fileSize: 1000, numParts: 4, expected: 248},
		{fileSize: 1000, numParts: 5, expected: 200},
		{fileSize: 1024, numParts: 4, expected: 256},
		{fileSize: 1024, numParts: 5, expected: 200},
		{fileSize: 1024, numParts: 3, expected: 336},
		{fileSize: 1024, numParts: 1, expected: 1024},
		{fileSize: 0, numParts: 1, expected: 0},
		{fileSize: 100, numParts: 3, expected: 32},
		{fileSize: 100, numParts: 7, expected: 8},
	}

	for _, tt := range tests {
		result := ChunkSize(tt.fileSize, tt.numParts)
		if result != tt.expected {
			t.Errorf("ChunkSize(%d, %d) = %d; expected %d", tt.fileSize, tt.numParts, result, tt.expected)
		}
	}
}

func TestChunkOffsets(t *testing.T) {
	tests := []struct {
		fileSize int64
		numParts int
		expected []int64
	}{
		{fileSize: 1000, numParts: 4, expected: []int64{0, 248, 496, 744}},
		{fileSize: 1000, numParts: 5, expected: []int64{0, 200, 400, 600, 800}},
		{fileSize: 1024, numParts: 4, expected: []int64{0, 256, 512, 768}},
		{fileSize: 1024, numParts: 5, expected: []int64{0, 200, 400, 600, 800}},
		{fileSize: 1024, numParts: 3, expected: []int64{0, 336, 672}},
		{fileSize: 1024, numParts: 1, expected: []int64{0}},
		{fileSize: 0, numParts: 1, expected: []int64{0}},
		{fileSize: 100, numParts: 3, expected: []int64{0, 32, 64}},
		{fileSize: 100, numParts: 7, expected: []int64{0, 8, 16, 24, 32, 40, 48}},
	}

	for _, tt := range tests {
		result := ChunkOffsets(tt.fileSize, tt.numParts)
		if !reflect.DeepEqual(result, tt.expected) {
			t.Errorf("ChunkOffsets(%d, %d) = %v; expected %v", tt.fileSize, tt.numParts, result, tt.expected)
		}
	}
}

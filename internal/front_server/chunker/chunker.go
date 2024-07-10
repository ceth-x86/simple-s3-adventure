package chunker

const wordSize = 8

func ChunkSize(fileSize int64, numParts int) int64 {
	// Size of each part without alignment consideration
	partSize := fileSize / int64(numParts)

	// Align the size of each part to be a multiple of the word size
	alignedPartSize := (partSize / int64(wordSize)) * int64(wordSize)

	return alignedPartSize
}

func ChunkOffsets(fileSize int64, numParts int) []int64 {
	chSize := ChunkSize(fileSize, numParts)
	offsets := make([]int64, numParts)
	offsets[0] = 0
	for i := 1; i < numParts; i++ {
		offsets[i] = offsets[i-1] + chSize
	}
	return offsets
}

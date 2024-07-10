package registry_service

import (
	"container/list"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"

	"simple-s3-adventure/pkg/logger"
)

const (
	fillFactor = 1.2 // must be greater than 1
	rounds     = 3
)

var (
	ErrChunkServerAlreadyRegistered = errors.New("chunk server already registered")
)

type ChunkServer struct {
	Address string
	size    int64
}

func (cs *ChunkServer) addSize(size int64) {
	atomic.AddInt64(&cs.size, size)
}

// ChunkAllocationMap is a map of file UUIDs to their parts.
type ChunkAllocationMap struct {
	chunks map[string][]*ChunkServer
	mu     sync.RWMutex
}

func NewChunkAllocationMap() *ChunkAllocationMap {
	return &ChunkAllocationMap{
		chunks: make(map[string][]*ChunkServer),
	}
}

func (c *ChunkAllocationMap) AddChunk(fileUUID string, servers []*ChunkServer) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.chunks[fileUUID] = servers
}

func (c *ChunkAllocationMap) GetChunkServers(fileUUID string) []*ChunkServer {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.chunks[fileUUID]
}

// ChunkServerRegistry is a catalog of chunk servers.
type ChunkServerRegistry struct {
	// chunkServerAddresses is a set of chunk server addresses. We use it to ensure the uniqueness.
	chunkServerAddresses map[string]struct{}

	// chunkServers is a round-robin list of chunk servers.
	chunkServers *list.List
	nextServer   *list.Element

	totalSize int64

	mu sync.RWMutex
}

func NewChunkServerRegistry() *ChunkServerRegistry {
	return &ChunkServerRegistry{
		chunkServerAddresses: make(map[string]struct{}),
		chunkServers:         list.New(),
	}
}

func (c *ChunkServerRegistry) AddChunkServer(url string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.chunkServerAddresses[url]; exists {
		return ErrChunkServerAlreadyRegistered
	}

	c.chunkServerAddresses[url] = struct{}{}
	c.chunkServers.PushBack(&ChunkServer{Address: url, size: 0})

	return nil
}

// AdjustSizes adjusts the sizes of the chunk servers.
func (c *ChunkServerRegistry) AdjustSizes(servers []*ChunkServer, sizes []int64, totalSize int64) {
	for i, size := range sizes {
		// atomic is used inside addSize()
		servers[i].addSize(size)
	}

	c.mu.Lock()
	c.totalSize += totalSize
	defer c.mu.Unlock()
}

// SelectUnderloadedChunkServers selects n underloaded chunk servers.
// An underloaded chunk server is a server whose size is less than the threshold.
// If there are not enough underloaded servers, it selects the servers with size greater than the threshold.
func (c *ChunkServerRegistry) SelectUnderloadedChunkServers(n int) []*ChunkServer {
	lg := logger.GetLogger()
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.chunkServerAddresses) < n {
		return nil
	}

	// chunkServersMap is used to check if the server is already selected
	chunkServersMap := make(map[string]struct{}, n)
	chunkServers := make([]*ChunkServer, 0, n)

	sizeThreshold := sizeThreshold(c.totalSize, int64(len(c.chunkServerAddresses)), fillFactor)
	lg.Info("Selecting servers",
		slog.Int64("totalSize", c.totalSize),
		slog.Int("numOfServers", len(c.chunkServerAddresses)),
		slog.Int64("threshold", sizeThreshold))

	// nextServer == nil if when server just launched
	if c.nextServer == nil {
		c.nextServer = c.chunkServers.Front()
	}
	startFromServer := c.nextServer.Value.(*ChunkServer).Address
	firstRound := true
	readyToGetOversized := false

	round := 0
	for len(chunkServers) < n || round < rounds {
		// end of the list, start from the beginning
		if c.nextServer == nil {
			c.nextServer = c.chunkServers.Front()
		}

		for ; c.nextServer != nil; c.nextServer = c.nextServer.Next() {
			address := c.nextServer.Value.(*ChunkServer).Address
			serverSize := c.nextServer.Value.(*ChunkServer).size
			if address == startFromServer {
				if !firstRound {
					// We have gone through all servers once, so now we are ready get servers with size > threshold
					readyToGetOversized = true
				}
				firstRound = false
			}

			// skip already selected servers
			if _, ok := chunkServersMap[address]; ok {
				continue
			}
			willBeSelected := serverSize < sizeThreshold || readyToGetOversized

			lg.Info("Checking server",
				slog.String("Address", address),
				slog.Int64("size", serverSize),
				slog.Bool("ready_to_get_oversized", readyToGetOversized),
				slog.Bool("result", willBeSelected),
				slog.Int("attempts", round),
			)

			if willBeSelected {
				chunkServers = append(chunkServers, c.nextServer.Value.(*ChunkServer))
				chunkServersMap[address] = struct{}{}
				if len(chunkServers) == n {
					return chunkServers
				}
			}
		}
		round++
	}
	return nil
}

// sizeThreshold calculates the threshold for the size of the chunk server.
// try to choose chunk servers with size less than threshold
func sizeThreshold(totalSize int64, numOfServers int64, fillFactor float64) int64 {
	threshold := int64(float64(totalSize/numOfServers) * fillFactor)
	// totalSize == 0
	if threshold == 0 {
		return 1
	}
	return threshold
}

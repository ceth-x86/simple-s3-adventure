package front_server

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

type chunkServer struct {
	address string
	size    int64
}

func (cs *chunkServer) addSize(size int64) {
	atomic.AddInt64(&cs.size, size)
}

// chunkAllocationMap is a map of file UUIDs to their parts.
type chunkAllocationMap struct {
	chunks map[string][]*chunkServer
	mu     sync.RWMutex
}

func (c *chunkAllocationMap) addChunk(fileUUID string, servers []*chunkServer) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.chunks[fileUUID] = servers
}

func (c *chunkAllocationMap) getChunkServers(fileUUID string) []*chunkServer {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.chunks[fileUUID]
}

// chunkServerRegistry is a catalog of chunk servers.
type chunkServerRegistry struct {
	// chunkServerAddresses is a set of chunk server addresses. We use it to ensure the uniqueness.
	chunkServerAddresses map[string]struct{}

	// chunkServers is a round-robin list of chunk servers.
	chunkServers *list.List
	nextServer   *list.Element

	totalSize int64

	mu sync.RWMutex
}

func (c *chunkServerRegistry) addChunkServer(url string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.chunkServerAddresses[url]; exists {
		return ErrChunkServerAlreadyRegistered
	}

	c.chunkServerAddresses[url] = struct{}{}
	c.chunkServers.PushBack(&chunkServer{address: url, size: 0})

	return nil
}

// adjustSizes adjusts the sizes of the chunk servers.
func (c *chunkServerRegistry) adjustSizes(servers []*chunkServer, sizes []int64, totalSize int64) {
	for i, size := range sizes {
		// atomic is used inside addSize()
		servers[i].addSize(size)
	}

	c.mu.Lock()
	c.totalSize += totalSize
	defer c.mu.Unlock()
}

// selectUnderloadedChunkServers selects n underloaded chunk servers.
// An underloaded chunk server is a server whose size is less than the threshold.
// If there are not enough underloaded servers, it selects the servers with size greater than the threshold.
func (c *chunkServerRegistry) selectUnderloadedChunkServers(n int) []*chunkServer {
	lg := logger.GetLogger()
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.chunkServerAddresses) < n {
		return nil
	}

	// chunkServersMap is used to check if the server is already selected
	chunkServersMap := make(map[string]struct{}, n)
	chunkServers := make([]*chunkServer, 0, n)

	sizeThreshold := sizeThreshold(c.totalSize, int64(len(c.chunkServerAddresses)), fillFactor)
	lg.Info("Selecting servers",
		slog.Int64("totalSize", c.totalSize),
		slog.Int("numOfServers", len(c.chunkServerAddresses)),
		slog.Int64("threshold", sizeThreshold))

	// nextServer == nil if when server just launched
	if c.nextServer == nil {
		c.nextServer = c.chunkServers.Front()
	}
	startFromServer := c.nextServer.Value.(*chunkServer).address
	firstRound := true
	readyToGetOversized := false

	round := 0
	for len(chunkServers) < n || round < rounds {
		// end of the list, start from the beginning
		if c.nextServer == nil {
			c.nextServer = c.chunkServers.Front()
		}

		for ; c.nextServer != nil; c.nextServer = c.nextServer.Next() {
			address := c.nextServer.Value.(*chunkServer).address
			serverSize := c.nextServer.Value.(*chunkServer).size
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
				slog.String("address", address),
				slog.Int64("size", serverSize),
				slog.Bool("ready_to_get_oversized", readyToGetOversized),
				slog.Bool("result", willBeSelected),
				slog.Int("attempts", round),
			)

			if willBeSelected {
				chunkServers = append(chunkServers, c.nextServer.Value.(*chunkServer))
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

package registry_service

import (
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChunkServerRegistry_AddChunkServer(t *testing.T) {
	registry := NewChunkServerRegistry()

	t.Run("Add new chunk server", func(t *testing.T) {
		err := registry.AddChunkServer("http://chunkserver1")
		assert.NoError(t, err)
		assert.Contains(t, registry.chunkServerAddresses, "http://chunkserver1")
	})

	t.Run("Add duplicate chunk server", func(t *testing.T) {
		err := registry.AddChunkServer("http://chunkserver1")
		assert.ErrorIs(t, err, ErrChunkServerAlreadyRegistered)
	})
}

func TestChunkServerRegistry_AdjustSizes(t *testing.T) {
	registry := NewChunkServerRegistry()

	server1 := &ChunkServer{Address: "http://chunkserver1", size: 0}
	server2 := &ChunkServer{Address: "http://chunkserver2", size: 0}
	registry.chunkServers.PushBack(server1)
	registry.chunkServers.PushBack(server2)

	sizes := []int64{100, 200}
	totalSize := int64(300)
	registry.AdjustSizes([]*ChunkServer{server1, server2}, sizes, totalSize)

	assert.Equal(t, int64(100), server1.size)
	assert.Equal(t, int64(200), server2.size)
	assert.Equal(t, int64(300), registry.totalSize)
}

func TestChunkServerRegistry_SelectUnderloadedChunkServers(t *testing.T) {
	registry := NewChunkServerRegistry()

	server1 := &ChunkServer{Address: "http://chunkserver1", size: 50}
	server2 := &ChunkServer{Address: "http://chunkserver2", size: 100}
	server3 := &ChunkServer{Address: "http://chunkserver3", size: 150}

	registry.chunkServerAddresses["http://chunkserver1"] = struct{}{}
	registry.chunkServerAddresses["http://chunkserver2"] = struct{}{}
	registry.chunkServerAddresses["http://chunkserver3"] = struct{}{}

	registry.chunkServers.PushBack(server1)
	registry.chunkServers.PushBack(server2)
	registry.chunkServers.PushBack(server3)

	registry.totalSize = 300

	t.Run("Select underloaded servers when enough available", func(t *testing.T) {
		servers := registry.SelectUnderloadedChunkServers(2)
		assert.Len(t, servers, 2)
		assert.Contains(t, servers, server1)
		assert.Contains(t, servers, server2)
	})

	t.Run("Select underloaded servers when enough available", func(t *testing.T) {
		servers := registry.SelectUnderloadedChunkServers(3)
		assert.Len(t, servers, 3)
		assert.Contains(t, servers, server1)
		assert.Contains(t, servers, server2)
		assert.Contains(t, servers, server3)
	})

	t.Run("Select when not enough available", func(t *testing.T) {
		servers := registry.SelectUnderloadedChunkServers(5)
		assert.Nil(t, servers)
	})
}

func TestChunkAllocationMap(t *testing.T) {
	cam := NewChunkAllocationMap()

	chunkServer1 := &ChunkServer{Address: "http://chunkserver1", size: 0}
	chunkServer2 := &ChunkServer{Address: "http://chunkserver2", size: 0}
	cam.AddChunk("file1", []*ChunkServer{chunkServer1, chunkServer2})

	t.Run("Get chunk servers for existing file", func(t *testing.T) {
		servers := cam.GetChunkServers("file1")
		assert.Len(t, servers, 2)
	})

	t.Run("Get chunk servers for non-existing file", func(t *testing.T) {
		servers := cam.GetChunkServers("file2")
		assert.Nil(t, servers)
	})
}

func TestSizeThreshold(t *testing.T) {
	t.Run("Calculate threshold with total size", func(t *testing.T) {
		threshold := sizeThreshold(300, 3, 1.2)
		assert.Equal(t, int64(120), threshold)
	})

	t.Run("Calculate threshold with zero total size", func(t *testing.T) {
		threshold := sizeThreshold(0, 3, 1.2)
		assert.Equal(t, int64(1), threshold)
	})
}

func TestChunkServer_AddSize(t *testing.T) {
	server := &ChunkServer{Address: "http://chunkserver1", size: 0}
	server.addSize(100)
	assert.Equal(t, int64(100), server.size)

	server.addSize(-50)
	assert.Equal(t, int64(50), server.size)
}

func TestChunkServerRegistry_AddChunkServerConcurrency(t *testing.T) {
	registry := NewChunkServerRegistry()

	var wg sync.WaitGroup
	serverCount := 100
	wg.Add(serverCount)

	for i := 0; i < serverCount; i++ {
		go func(i int) {
			defer wg.Done()
			url := "http://chunkserver" + strconv.Itoa(i)
			_ = registry.AddChunkServer(url)
		}(i)
	}

	wg.Wait()
	assert.Equal(t, serverCount, len(registry.chunkServerAddresses))
}

func TestChunkServerRegistry_AdjustSizesConcurrency(t *testing.T) {
	registry := NewChunkServerRegistry()

	server1 := &ChunkServer{Address: "http://chunkserver1", size: 0}
	server2 := &ChunkServer{Address: "http://chunkserver2", size: 0}
	registry.chunkServers.PushBack(server1)
	registry.chunkServers.PushBack(server2)

	var wg sync.WaitGroup
	sizeUpdates := 100
	wg.Add(sizeUpdates * 2)

	for i := 0; i < sizeUpdates; i++ {
		go func() {
			defer wg.Done()
			registry.AdjustSizes([]*ChunkServer{server1}, []int64{10}, 10)
		}()

		go func() {
			defer wg.Done()
			registry.AdjustSizes([]*ChunkServer{server2}, []int64{20}, 20)
		}()
	}

	wg.Wait()
	assert.Equal(t, int64(1000), server1.size)
	assert.Equal(t, int64(2000), server2.size)
	assert.Equal(t, int64(3000), registry.totalSize)
}

func TestChunkServerRegistry_SelectUnderloadedChunkServersWithWrapAround(t *testing.T) {
	registry := NewChunkServerRegistry()

	server1 := &ChunkServer{Address: "http://chunkserver1", size: 50}
	server2 := &ChunkServer{Address: "http://chunkserver2", size: 60}
	server3 := &ChunkServer{Address: "http://chunkserver3", size: 70}

	registry.chunkServerAddresses["http://chunkserver1"] = struct{}{}
	registry.chunkServerAddresses["http://chunkserver2"] = struct{}{}
	registry.chunkServerAddresses["http://chunkserver3"] = struct{}{}

	registry.chunkServers.PushBack(server1)
	registry.chunkServers.PushBack(server2)
	registry.chunkServers.PushBack(server3)

	registry.totalSize = 180
	registry.nextServer = registry.chunkServers.Front().Next() // Start from the second server

	servers := registry.SelectUnderloadedChunkServers(2)
	assert.Len(t, servers, 2)
	assert.Equal(t, server2, servers[0])
	assert.Equal(t, server3, servers[1])
}

func TestChunkServerRegistry_SelectAfterAddingNewServer(t *testing.T) {
	registry := NewChunkServerRegistry()

	server1 := &ChunkServer{Address: "http://chunkserver1", size: 50}
	server2 := &ChunkServer{Address: "http://chunkserver2", size: 50}
	server3 := &ChunkServer{Address: "http://chunkserver3", size: 0}

	registry.chunkServerAddresses["http://chunkserver1"] = struct{}{}
	registry.chunkServerAddresses["http://chunkserver2"] = struct{}{}
	registry.chunkServerAddresses["http://chunkserver3"] = struct{}{}

	registry.chunkServers.PushBack(server1)
	registry.chunkServers.PushBack(server2)
	registry.chunkServers.PushBack(server3)

	registry.totalSize = 100
	registry.nextServer = registry.chunkServers.Front().Next() // Start from the second server

	servers := registry.SelectUnderloadedChunkServers(3)
	assert.Len(t, servers, 3)

	assert.Equal(t, server3, servers[0])
	assert.Equal(t, server2, servers[1])
	assert.Equal(t, server1, servers[2])
}

func TestChunkServerRegistry_SelectUnderloadedChunkServersConcurrency(t *testing.T) {
	registry := NewChunkServerRegistry()

	serverCount := 10
	for i := 0; i < serverCount; i++ {
		server := &ChunkServer{Address: "http://chunkserver" + string(rune(i)), size: int64(i * 10)}
		registry.chunkServerAddresses[server.Address] = struct{}{}
		registry.chunkServers.PushBack(server)
		registry.totalSize += int64(i * 10)
	}

	var wg sync.WaitGroup
	selectionCount := 100
	wg.Add(selectionCount)

	for i := 0; i < selectionCount; i++ {
		go func() {
			defer wg.Done()
			servers := registry.SelectUnderloadedChunkServers(5)
			assert.Len(t, servers, 5)
		}()
	}

	wg.Wait()
}

func TestChunkServerRegistry_AddAndSelectConcurrency(t *testing.T) {
	registry := NewChunkServerRegistry()

	var wg sync.WaitGroup
	serverCount := 50
	selectionCount := 100
	wg.Add(serverCount + selectionCount)

	for i := 0; i < serverCount; i++ {
		go func(i int) {
			defer wg.Done()
			url := "http://chunkserver" + string(rune(i))
			_ = registry.AddChunkServer(url)
		}(i)
	}

	for i := 0; i < selectionCount; i++ {
		go func() {
			defer wg.Done()
			servers := registry.SelectUnderloadedChunkServers(5)
			if len(registry.chunkServerAddresses) >= 5 {
				assert.Len(t, servers, 5)
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, serverCount, len(registry.chunkServerAddresses))
}

func TestChunkServerRegistry_AdjustAndSelectConcurrency(t *testing.T) {
	registry := NewChunkServerRegistry()

	server1 := &ChunkServer{Address: "http://chunkserver1", size: 0}
	server2 := &ChunkServer{Address: "http://chunkserver2", size: 0}
	registry.chunkServerAddresses[server1.Address] = struct{}{}
	registry.chunkServerAddresses[server2.Address] = struct{}{}
	registry.chunkServers.PushBack(server1)
	registry.chunkServers.PushBack(server2)

	var wg sync.WaitGroup
	adjustmentCount := 100
	selectionCount := 100
	wg.Add(adjustmentCount + selectionCount)

	for i := 0; i < adjustmentCount; i++ {
		go func() {
			defer wg.Done()
			registry.AdjustSizes([]*ChunkServer{server1, server2}, []int64{10, 20}, 30)
		}()
	}

	for i := 0; i < selectionCount; i++ {
		go func() {
			defer wg.Done()
			servers := registry.SelectUnderloadedChunkServers(2)
			assert.Len(t, servers, 2)
		}()
	}

	wg.Wait()
	assert.Equal(t, int64(1000), server1.size)
	assert.Equal(t, int64(2000), server2.size)
	assert.Equal(t, int64(3000), registry.totalSize)
}

func TestChunkServerRegistry_AddChunkServerHighConcurrency(t *testing.T) {
	registry := NewChunkServerRegistry()

	var wg sync.WaitGroup
	serverCount := 1000
	wg.Add(serverCount)

	for i := 0; i < serverCount; i++ {
		go func(i int) {
			defer wg.Done()
			url := "http://chunkserver" + string(rune(i))
			_ = registry.AddChunkServer(url)
		}(i)
	}

	wg.Wait()
	assert.Equal(t, serverCount, len(registry.chunkServerAddresses))
}

package front_service_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"simple-s3-adventure/internal/front_server/front_service"
	"simple-s3-adventure/internal/front_server/registry_service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type FrontServiceSuite struct {
	suite.Suite
	server *httptest.Server
	fs     *front_service.FrontService
}

func (suite *FrontServiceSuite) SetupTest() {
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/get" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("chunk data"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// Create a mock registry and allocation map
	registry := &registry_service.ChunkServerRegistry{}
	allocationMap := registry_service.NewChunkAllocationMap()
	chunkServer := registry_service.ChunkServer{Address: suite.server.URL}
	allocationMap.AddChunk("test-uuid", []*registry_service.ChunkServer{&chunkServer})
	suite.fs = front_service.NewFrontService(registry, allocationMap)
}

func (suite *FrontServiceSuite) TearDownTest() {
	suite.server.Close()
}

func (suite *FrontServiceSuite) TestCopyChunks() {
	var buffer bytes.Buffer
	n, err := suite.fs.CopyChunks("test-uuid", &buffer)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(len("chunk data")), n)
	assert.Equal(suite.T(), "chunk data", buffer.String())
}

func TestFrontServiceSuite(t *testing.T) {
	suite.Run(t, new(FrontServiceSuite))
}

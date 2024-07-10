package download_service

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"simple-s3-adventure/internal/front_server/registry_service"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// Suite for DownloadService tests
type DownloadServiceSuite struct {
	suite.Suite
	service *DownloadService
	server  *httptest.Server
}

func (suite *DownloadServiceSuite) SetupTest() {
	// Create a test HTTP server
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/get" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("chunk data"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// Create ChunkCopyManager instance
	chunkServers := []*registry_service.ChunkServer{
		{Address: suite.server.URL},
	}
	suite.service = NewDownloadService("test-uuid", chunkServers)
}

func (suite *DownloadServiceSuite) TearDownTest() {
	suite.server.Close()
}

func (suite *DownloadServiceSuite) TestCopyChunksSuccess() {
	var buffer bytes.Buffer
	n, err := suite.service.CopyChunks(&buffer)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(len("chunk data")), n)
	assert.Equal(suite.T(), "chunk data", buffer.String())
}

func TestChunkCopyManagerSuite(t *testing.T) {
	suite.Run(t, new(DownloadServiceSuite))
}

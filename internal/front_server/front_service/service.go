package front_service

import (
	"log/slog"
	"net/http"
	"simple-s3-adventure/internal/front_server/registry_service"
	"simple-s3-adventure/pkg/logger"
)

type FrontService struct {
	registry      *registry_service.ChunkServerRegistry
	allocationMap *registry_service.ChunkAllocationMap
	httpClient    *http.Client
	logger        *slog.Logger
}

func NewFrontService(registry *registry_service.ChunkServerRegistry, allocationMap *registry_service.ChunkAllocationMap) *FrontService {
	return &FrontService{
		registry:      registry,
		allocationMap: allocationMap,
		httpClient:    &http.Client{},
		logger:        logger.GetLogger(),
	}
}

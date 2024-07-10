package main

import (
	"simple-s3-adventure/internal/chunk_server/api"
	"simple-s3-adventure/internal/chunk_server/service"
)

func main() {
	api.StartServer(service.NewServerConfig())
}

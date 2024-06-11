package main

import (
	"simple-s3-adventure/internal/chunk_server"
)

func main() {
	chunk_server.StartServer(chunk_server.NewServerConfig())
}

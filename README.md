# Amazon S3 rival plan

I've decided to take on Amazon S3 and have figured out how to build the ultimate object storage service.

Conditions:

- Several servers for storing the chunks and one server to rule them all.
- REST API for uploading and downloading files.
- The file should be split into chunks and stored on different servers.
- Storage servers can be added at any time, but no, you can't remove it.
- The storage load is evenly distributed among the servers.

## Architecture

Refer to the document [Architecture](docs/architecture.md) to understand the main line of thinking and the reasons behind the architectural decisions made.

## Build and run

This command build and launch a cluster consisting of one frontend server and ten chunk servers:

```sh
docker-compose up
```

Upload file:

```sh
curl -X PUT -F file=@example.pdf 'http://localhost:13090/put'
```

Download file:

```sh
curl -X GET 'http://localhost:13090/get?uuid=69d973de-c7ba-4856-9e54-773bb0e58546' > example_result.pdf
```

## TODO:

Frontend server:

[ ] A set of endpoints for collecting statistics from the frontend server: for example, data distribution across nodes.

Chunk server:

[ ] Checking for available space before uploading a chunk.

Chunk server and frontend server:

[ ] Health and live checks.

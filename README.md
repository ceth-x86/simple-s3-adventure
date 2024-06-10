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


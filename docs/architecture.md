# Architecture

We have two applications: Front-Server and Chunk-Server.

The Front-Server is responsible for:

- Handling clients’ REST requests.
- Splitting data into chunks and assembling files from chunks.
- Storing metadata about files and their chunks.

The Chunk-Server is responsible for:

- Providing an API for reading and writing chunks.

Architectural decisions:

- When splitting a file into chunks, we need to avoid large memory allocations.
- What should be used for communication between the API-Server and Chunk-Server? I would like to use gRPC, but to simplify, I will start with REST API.
- How to store metadata? It might be worth using a database, but to simplify, we will start with in-memory storage. Using a database in the future will allow us to switch to a configuration with multiple Front servers.
- When adding a new Chunk-Server, chunks need to be redistributed among servers.

Open questions:

- Choosing Chunk servers at the time of writing a file.
- What to do if one of the Chunk servers returns an error when writing a chunk? Should we return an error to the client or try to write the chunk to another server?л ошибкой при записи чанка? Возвращать ошибку клиенту или пытаться записать чанк на другой сервер?

Things to do:

- Metadata.
- Checksums.
- Replication and data recovery in case of Chunk-Server node failure.

Out of scope:

- Authorization and authentication.

## What is our chunk size?

With a maximum file size of 10GB and 6 servers, the maximum chunk size will be 1.67GB. To avoid creating a large number of small chunks, we will introduce a minimum chunk size of 1MB. This is a deliberate decision:

- If a user wants to upload a file of 1MB, they will receive 1 chunk.
- If a user wants to upload a file of 3MB, they will receive 3 chunks.

## How will we avoid large memory allocations when working with files?

We will rely on working with standard interfaces `io.Reader` and `io.Writer`.

When writing a file:

- We receive a file upload request on the API server.
- We create up to 6 io.Reader using the io.NewSectionReader function. 
- We use them to transfer data to the Chunk servers.

When downloading a file:

- We receive a file download request on the API server.
- We have a sync.Map that stores an ordered list of chunk servers where the file is stored.
- We make up to 6 requests to download the chunks.
- We sequentially read the data from the chunk servers’ responses into the API server’s response.

## How will we select Chunk servers to store chunks?

On the Front server, we have a list of all Chunk servers with the amount of data on them. We could select the 6 servers that contain the least amount of data. Here, a data structure like a heap will help us.

But we might have multiple competing requests, and if we follow this strategy, all these requests will hit the same chunk servers. This will result in the servers with the least amount of data becoming heavily overloaded with requests.

We could store not only the volume of data on each chunk server but also an indicator (or the count) of requests that it is currently processing (infly requests). We should prioritize selecting servers with fewer ongoing requests and, secondarily, those with less data.

## How the front server know about all the chunk servers?

The address of the front server is a parameter of the chunk server. When the chunk server starts, it registers with the front server. During this process, the front server receives information about the amount of data on the chunk server.

Each front server must maintain an endpoint that returns information about its availability. If a chunk server does not respond, the front server removes it from the list of available servers and stops redirecting requests to that server.

## What happens if a chunk server crashes and then will be restarted

Upon restarting, the chunk server can scan its directory and send information about all chunks to the front server. The front server should update the information about the amount of data on the chunk server.

If there is a large amount of data on the chunk server, this process can take a considerable amount of time. It might be worthwhile to store this metadata in a separate file.

While the chunk server was unavailable, data associated with this key might have changed. In this case, the chunk server contains incorrect data. Therefore, for each chunk, we will store a checksum both on the chunk server and on the front server. If the front server receives information about a chunk with an outdated checksum, it will consider it invalid.

## What happens if the front server crashes?

We will lose all data. The chunk servers still have information about the chunks, but we will lose information on how to reconstruct files from these chunks. To solve this problem, we should use some external storage, such as a database.

Additionally, our service will become unavailable at that moment (it will stop accepting new requests). To increase the availability of our solution, we should consider running multiple front servers with a shared database. Requests should be load-balanced among them.

## Do we need to balance when adding a new chunk server?

I haven’t decided on this yet. But if we do, we need to consider that:

- Balancing should not interfere with the existing requests.

How will we select Chunk servers for balancing?

- We take the chunk server with the largest amount of data.
- We copy a chunk from it to the server with the smallest amount of data.
- We repeat until the difference in data volume between the servers becomes less than a certain threshold. With a threshold greater than or equal to 2 * chunk size, such data copying will lead to volume-based balancing.
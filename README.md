Syncinator is a cloud-based file storage solution that lets users synchronize files between their local machine and a remote server. It’s built on a client–server model using gRPC for communication.

## Architecture

### Components

Syncinator is composed of these parts:

- **Client**  
  Communicates with the server over gRPC to upload, download, and synchronize files.
- **BlockStore**  
  Holds file data in fixed-size chunks called blocks, each identified by a unique hash. Provides block retrieval by hash.
- **MetaStore**  
  Maintains file metadata, linking filenames to their constituent blocks. Keeps track of all BlockStore instances and which blocks they store. Designed to scale to large datasets.

Metadata consistency across MetaStore replicas is ensured via the RAFT consensus algorithm, so the service remains reliable even if some nodes fail. BlockStores are distributed using consistent hashing: each store’s address is hashed (SHA-256) to form a ring, and blocks are mapped onto that ring.

### Core Concepts

- **File Division**  
  Files are split into uniform-size blocks (except the final block). Each block’s SHA-256 hash is recorded in a hash list for the file.
- **Version Control**  
  Every file has a strictly increasing version number. Clients compare versions to detect stale views.
- **Base Directory**  
  A designated folder on the client that mirrors the cloud state. Only files inside this directory are watched and synced.
- **Local Index (`index.db`)**  
  Stores the last-known snapshot of the base directory’s contents for quick change detection.

### Sync Engine

The sync engine uses a three-tree model to decide what to upload, download, or merge:

- **Remote Tree**  
  The current state of your files on the server.
- **Local Tree**  
  The last-observed state on disk, reconstructed from `index.db`.
- **Synced Tree**  
  The last fully synchronized state between local and remote—used as the merge base to distinguish local edits from remote updates.

By comparing these three views, Syncinator determines which changes originated locally or remotely, and applies updates accordingly.

### Configuration

The server reads a JSON config file listing all MetaStore and BlockStore endpoints. You can experiment with different topologies by editing [config.json](config.json).

## Synchronization Algorithm

1. Client loads `index.db` to reconstruct the Local Tree.
2. Scans the base directory, computes new hash lists, and updates the Local Tree for any added, changed, or removed files.
3. Saves this updated view as the Synced Tree.
4. Fetches the Remote Tree from the server.
5. Uses the Synced Tree to determine whether differences are local or remote.
6. On conflicts, the server accepts the first client’s changes and rejects later conflicting writes.
7. Downloads files when the remote version is newer; uploads local changes otherwise.

## Setup

```bash
# install Protocol Compiler Plugins for Go
$ go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
$ go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

# Add the Go binaries to your PATH
$ export PATH="$PATH:$(go env GOPATH)/bin"

# Install dependencies
$ go mod tidy

$ protoc --proto_path=. \
    --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    pkg/syncinator/Syncinator.proto

# Run the tests
$ make test
# Run specific tests
$ make TEST_REGEX=TestSpecific
# Clean up
$ make clean
```

## Usage

1. **Make the helper scripts executable**

   ```bash
   chmod +x run_block_server.sh run_meta_server.sh run_syncinator.sh
   ```

2. **Launch the BlockStore**

   ```bash
   ./run_block_server.sh
   ```

   Starts a BlockStore listening for block read/write requests.

3. **Launch the MetaStore (with RAFT)**

   ```bash
   ./run_meta_server.sh
   ```

   Starts the RAFT-backed MetaStore to coordinate metadata.

4. **Start the Syncinator client**
   ```bash
   ./run_syncinator.sh <local_folder>
   ```
   Replace `<local_folder>` with the directory you want to keep in sync. You can run multiple clients pointing at different folders.

---

**Example**

```bash
# 1. Make scripts executable
chmod +x run_block_server.sh run_meta_server.sh run_syncinator.sh

# 2. Start services in separate terminals
./run_block_server.sh       # Terminal A
./run_meta_server.sh        # Terminal B

# 3. Sync multiple folders
./run_syncinator.sh folderA # Terminal C
./run_syncinator.sh folderB # Terminal D
./run_syncinator.sh folderC # Terminal E
```

## Testing

Syncinator includes a `RaftTestingInterface` for simulating crashes and partitions:

- `Crash()`
- `MakeServerUnreachableFrom()`
- `Restore()`

These let you verify correct RAFT behavior under failure conditions. See tests in [pkg/test/raft_test.go](pkg/test/raft_test.go) and [pkg/test/raft_client_test.go](pkg/test/raft_client_test.go).

## References

1. [gRPC](https://grpc.io/)
2. [RAFT Consensus Algorithm](https://raft.github.io)
3. [Consistent Hashing](https://en.wikipedia.org/wiki/Consistent_hashing)
4. [Student’s Guide to RAFT](https://thesquareplanet.com/blog/students-guide-to-raft/)
5. [Instructor’s Guide to RAFT](https://thesquareplanet.com/blog/instructors-guide-to-raft/)
6. [Effective Go](https://golang.org/doc/effective_go.html)
7. [Go by Example](https://gobyexample.com/)
8. [GoDoc](https://pkg.go.dev/)

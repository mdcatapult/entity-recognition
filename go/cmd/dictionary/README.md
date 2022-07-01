# Dictionary

This is a gRPC server which connects to redis. The server receives a stream of tokens (redis keys) and looks them up in redis, returning a stream of recognised entities (redis values).

The main handler function for gRPC requests is defined in `../../proto/services.proto` to be GetStream in recognizer.go. gRPC will call this under the hood once to set up the input (Snippet) and output (Entity) streams.

This service can be configured using yml. The yml must be located in `./config/dictionary.yml`, relative from the NER project root. See the existing config for examples. 

### Running

This service can be run using Go or Docker: 

- `go build ./... && ./dictionary`
- The dockerfile is intended to be run from the NER project root: `docker build -f go/cmd/dictionary/Dockerfile -t dictionary-test .`

### Testing

`go test ./...`

Requires a redis instance running on port 6379. Either `docker-compose up` from the NER project root, or run `docker run -p 6379:6379 redis:latest`.



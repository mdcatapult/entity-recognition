# Entity recognition

This [*monorepo*](https://www.atlassian.com/git/tutorials/monorepos) contains all MDC entity recognition software, configuration, documentation, and so on.

The intention of this work is to modernise chemical entity recognition via up to date architecture, state of the art models, and brute force.

## Generate protobufs
Go:
```bash
protoc --proto_path=./proto --go_out=./go/gen/pb \
  --go_opt=paths=source_relative --go-grpc_out=./go/gen/pb \
  --go-grpc_opt=paths=source_relative ./proto/*.proto
```
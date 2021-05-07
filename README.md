# Entity recognition

This [*monorepo*](https://www.atlassian.com/git/tutorials/monorepos) contains all MDC entity recognition software, configuration, documentation, and so on.

The intention of this work is to modernise chemical entity recognition via up to date architecture, state of the art models, and brute force.

## Run
```bash
docker-compose up -d
go run go/cmd/regexer/main.go --config config.example.yml
# open another terminal in the same directory, then:
go run go/cmd/dictionary/main.go --config config.example.yml
# open another terminal in the same directory, then:
go run go/cmd/rest-api/main.go --config config.example.yml
```

## Test
Grab some html from a website (ctrl+U in chrome). Make a post request to `localhost:8080/html/text`, `localhost:8080/html/tokens`, or `localhost:8080/html/entities` with the html in the body of the request.

## Generate protobufs
Go:
```bash
protoc --proto_path=./proto --go_out=./go/gen/pb \
  --go_opt=paths=source_relative --go-grpc_out=./go/gen/pb \
  --go-grpc_opt=paths=source_relative ./proto/*.proto
```
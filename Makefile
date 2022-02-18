PROTOC := $(shell command -v protoc 2> /dev/null)
MOCKERY := $(shell command -v mockery 2> /dev/null)
GO := $(shell command -v go 2> /dev/null)
PROTO_GEN_GO := $(shell command -v protoc-gen-go 2> /dev/null)
PROTO_GEN_GO_GRPC := $(shell command -v protoc-gen-go-grpc 2> /dev/null)

# Makes a copy of all the example config to put gitignored config files in the default location.
.PHONY: config
config:
	for FILE in $(shell find config/* -name *.example.*); do cp $$FILE $$(echo $$FILE | sed 's/\.example//'); done

# Check for go, error if not present and link to install instructions.
.PHONY: require-go
require-go:
ifndef GO
	$(error "install go https://golang.org/doc/install")
endif

# Builds all the go binaries and places them in the bin folder.
.PHONY: build
build: require-go
	for FOLDER in ./go/cmd/*; do go build -o bin/ $$FOLDER/...; done

# Check for mockery, error if not present and link to install instructions.
.PHONY: require-mockery
require-mockery:
ifndef MOCKERY
	$(error "install mockery https://github.com/vektra/mockery#go-get")
endif

# Generates all mocks with mockery.
.PHONY: mocks
mocks: require-mockery
	mockery --all --case underscore --output ./go/gen/mocks --keeptree --dir go

# Check for protoc and plugins, error if not present and link to install instructions.
.PHONY: require-protoc
require-protoc:
ifndef PROTOC
	$(error "install protoc https://grpc.io/docs/protoc-installation/")
endif
ifndef PROTO_GEN_GO
	$(error "install golang plugins https://grpc.io/docs/languages/go/quickstart/")
endif
ifndef PROTO_GEN_GO_GRPC
	$(error "install golang plugins https://grpc.io/docs/languages/go/quickstart/")
endif

# Generates all protobuf files.
.PHONY: proto
proto: require-protoc
	protoc --proto_path=./proto --go_out=./go/gen/pb \
      --go_opt=paths=source_relative --go-grpc_out=./go/gen/pb \
      --go-grpc_opt=paths=source_relative ./proto/*.proto

.PHONY: run
run: build
	@./scripts/run.sh

.PHONY: format
format: require-go
	find ./go -type f -name '*.go' -not -path "./go/gen/*" -exec ./scripts/goimports.sh {} \;

.PHONY: docs
docs:
	cd go/cmd/recognition-api; \
	swagger generate spec -o ./swagger.json --scan-models; \
	swagger serve -F=swagger swagger.json

.PHONY: api-test
api-test:
	@./scripts/test.sh
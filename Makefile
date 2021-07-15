.PHONY: config
config:
	for FILE in config/*.example.*; do cp $$FILE $$(echo $$FILE | sed 's/\.example//'); done

.PHONY: build
build:
	for FOLDER in ./go/cmd/*; do go build -o bin/ $$FOLDER/...; done

.PHONY: mocks
mocks:
	mockery --all --case underscore --output ./go/gen/mocks

.PHONY: proto
proto:
	protoc --proto_path=./proto --go_out=./go/gen/pb \
      --go_opt=paths=source_relative --go-grpc_out=./go/gen/pb \
      --go-grpc_opt=paths=source_relative ./proto/*.proto
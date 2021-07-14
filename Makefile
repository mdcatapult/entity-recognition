.PHONY: config
config:
	for FILE in config/*.example.*; do cp $$FILE $$(echo $$FILE | sed 's/\.example//'); done

.PHONY: build
build:
	for FOLDER in ./go/cmd/*; do go build -o bin/ $$FOLDER/...; done
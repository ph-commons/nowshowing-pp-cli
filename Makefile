.PHONY: build test lint install clean

BIN_EXT := $(if $(filter windows,$(shell go env GOOS)),.exe,)

build:
	go build -o bin/nowshowing-pp-cli$(BIN_EXT) ./cmd/nowshowing-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/nowshowing-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/nowshowing-pp-mcp$(BIN_EXT) ./cmd/nowshowing-pp-mcp

install-mcp:
	go install ./cmd/nowshowing-pp-mcp

build-all: build build-mcp

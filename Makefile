BINARY_SERVER=bin/server
BINARY_CLI=bin/taskctl
GO=go
GOFLAGS=-v

.PHONY: all build run test bench lint clean tidy

all: build

build:
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -o $(BINARY_SERVER) ./cmd/server
	$(GO) build $(GOFLAGS) -o $(BINARY_CLI) ./cmd/taskctl

run:
	$(GO) run ./cmd/server

test:
	$(GO) test -race -count=1 ./...

test/integration:
	$(GO) test -race -count=1 -v ./tests/integration/...

bench:
	$(GO) test -bench=. -benchmem -count=3 ./tests/benchmark/...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
	$(GO) clean -testcache

tidy:
	$(GO) mod tidy

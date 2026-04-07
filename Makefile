BINARY := jit
BIN_DIR := bin
DIST_DIR := dist
MAIN_PKG := .
PKG := ./...

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: help fmt test build clean snapshot release tidy

help:
	@echo "Targets:"
	@echo "  make fmt       - gofmt all Go files"
	@echo "  make test      - run go test ./..."
	@echo "  make build     - build local binary to ./bin/jit"
	@echo "  make snapshot  - goreleaser snapshot build"
	@echo "  make release   - goreleaser release"
	@echo "  make tidy      - go mod tidy"
	@echo "  make clean     - remove build artifacts"

fmt:
	@find . -name '*.go' -type f -print0 | xargs -0 gofmt -w

test:
	go test $(PKG)

build:
	@mkdir -p $(BIN_DIR)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(MAIN_PKG)

snapshot:
	goreleaser release --snapshot --clean

release:
	goreleaser release --clean

tidy:
	go mod tidy

clean:
	rm -rf $(BIN_DIR) $(DIST_DIR)

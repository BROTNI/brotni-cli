BINARY     := brotni
MODULE     := github.com/BROTNI/brotni-cli
CMD        := ./cmd/brotni

VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X $(MODULE)/internal/commands.Version=$(VERSION) \
	-X $(MODULE)/internal/commands.Commit=$(COMMIT) \
	-X $(MODULE)/internal/commands.BuildDate=$(BUILD_DATE)

.PHONY: all build clean test lint fmt vet tidy install cross help

all: build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

install:
	go install -ldflags "$(LDFLAGS)" $(CMD)

test:
	go test ./... -v -race -coverprofile=coverage.out

test-short:
	go test ./... -short

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

cross:
	GOOS=linux   GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64 $(CMD)
	GOOS=linux   GOARCH=arm64  go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-arm64 $(CMD)
	GOOS=darwin  GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64 $(CMD)
	GOOS=darwin  GOARCH=arm64  go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64 $(CMD)
	GOOS=windows GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe $(CMD)

clean:
	rm -f $(BINARY) coverage.out
	rm -rf dist/

help:
	@echo "Available targets:"
	@echo "  build       Build the brotni binary"
	@echo "  install     Install the brotni binary to GOPATH/bin"
	@echo "  test        Run tests with race detector and coverage"
	@echo "  test-short  Run tests without long-running cases"
	@echo "  lint        Run golangci-lint"
	@echo "  fmt         Format Go source files"
	@echo "  vet         Run go vet"
	@echo "  tidy        Run go mod tidy"
	@echo "  cross       Build for all supported platforms into dist/"
	@echo "  clean       Remove build artifacts"

.PHONY: build test test-verbose test-coverage clean install dev lint fmt release snapshot

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

build:
	go build -ldflags="$(LDFLAGS)" -o bin/potus ./cmd/potus

test:
	go test -race ./...

test-verbose:
	go test -v -race ./...

test-coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/ dist/ coverage.out coverage.html

install:
	go install -ldflags="$(LDFLAGS)" ./cmd/potus

dev:
	go run ./cmd/potus

lint:
	golangci-lint run

fmt:
	go fmt ./...
	goimports -w .

# Release with GoReleaser
release:
	@if [ -z "$(VERSION)" ]; then \
		echo "VERSION is required. Usage: make release VERSION=v1.0.0"; \
		exit 1; \
	fi
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)
	goreleaser release --clean

# Create a snapshot release (no git tag required)
snapshot:
	goreleaser release --snapshot --clean

# Build for all platforms manually
build-all: clean
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/potus-darwin-amd64 ./cmd/potus
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/potus-darwin-arm64 ./cmd/potus
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/potus-linux-amd64 ./cmd/potus
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/potus-linux-arm64 ./cmd/potus
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/potus-windows-amd64.exe ./cmd/potus

.DEFAULT_GOAL := build

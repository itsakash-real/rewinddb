# Nimbi Makefile
# Usage: make [target]

BINARY      := rw
MODULE      := github.com/itsakash-real/nimbi
CMD_PKG     := ./cmd/rw
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME  := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS     := -s -w \
               -X main.Version=$(VERSION) \
               -X main.BuildTime=$(BUILD_TIME) \
               -X main.GoVersion=$(shell go version | awk '{print $$3}')

GOFLAGS     := -trimpath
INSTALL_DIR ?= /usr/local/bin
DIST_DIR    := dist
COVERAGE    := coverage.out

.PHONY: all build test bench lint install uninstall release clean fmt vet \
        completions man snapshot

all: build

## ── Build ────────────────────────────────────────────────────────────────────

build:
	@echo "→ Building $(BINARY) $(VERSION)"
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY) $(CMD_PKG)

build-all:
	@echo "→ Cross-compiling for all targets"
	GOOS=linux   GOARCH=amd64  go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY)-linux-amd64   $(CMD_PKG)
	GOOS=linux   GOARCH=arm64  go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY)-linux-arm64   $(CMD_PKG)
	GOOS=darwin  GOARCH=amd64  go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY)-darwin-amd64  $(CMD_PKG)
	GOOS=darwin  GOARCH=arm64  go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY)-darwin-arm64  $(CMD_PKG)
	GOOS=windows GOARCH=amd64  go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY)-windows-amd64.exe $(CMD_PKG)

## ── Test ─────────────────────────────────────────────────────────────────────

test:
	@echo "→ Running tests (race + cover)"
	go test ./... -race -coverprofile=$(COVERAGE) -covermode=atomic -timeout=120s
	go tool cover -func=$(COVERAGE) | tail -1

test-short:
	go test ./... -short -race -timeout=60s

cover: test
	go tool cover -html=$(COVERAGE) -o coverage.html
	@echo "→ Coverage report: coverage.html"

bench:
	@echo "→ Running benchmarks"
	go test -tags bench ./bench/... -bench=. -benchmem -benchtime=3s -timeout=300s

## ── Quality ──────────────────────────────────────────────────────────────────

lint:
	@echo "→ Running golangci-lint"
	golangci-lint run ./... --timeout=5m

fmt:
	gofmt -w -s .
	goimports -w .

vet:
	go vet ./...

staticcheck:
	staticcheck ./...

## ── Install / Uninstall ──────────────────────────────────────────────────────

install: build
	@echo "→ Installing $(BINARY) to $(INSTALL_DIR)"
	install -m 0755 $(BINARY) $(INSTALL_DIR)/$(BINARY)

uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY)

## ── Distribution ─────────────────────────────────────────────────────────────

release:
	@echo "→ Running goreleaser release"
	goreleaser release --clean

snapshot:
	@echo "→ Running goreleaser snapshot (no publish)"
	goreleaser release --snapshot --clean

## ── Completions ──────────────────────────────────────────────────────────────

completions: build
	@echo "→ Generating shell completions"
	@mkdir -p completions
	./$(BINARY) completion bash > completions/$(BINARY).bash
	./$(BINARY) completion zsh  > completions/_$(BINARY)
	./$(BINARY) completion fish > completions/$(BINARY).fish
	./$(BINARY) completion powershell > completions/$(BINARY).ps1
	@echo "  completions/ directory populated"

## ── Man pages ────────────────────────────────────────────────────────────────

man: build
	@echo "→ Generating man pages"
	@mkdir -p man/man1
	./$(BINARY) man ./man/man1
	@echo "  man/man1/ populated"

man-install: man
	install -d /usr/local/share/man/man1
	install -m 0644 man/man1/*.1 /usr/local/share/man/man1/
	mandb 2>/dev/null || true

## ── Integration Tests ────────────────────────────────────────────────────────

test-integration:
	@echo "→ Running integration tests"
	go test -tags integration -v -race -timeout=120s ./tests/integration/

test-integration-bench:
	@echo "→ Running integration benchmarks"
	go test -tags integration -bench=. -benchmem -benchtime=3s \
		-timeout=300s ./tests/integration/

test-all: test test-integration


## ── Clean ────────────────────────────────────────────────────────────────────

clean:
	@echo "→ Cleaning artifacts"
	rm -f $(BINARY) $(COVERAGE) coverage.html
	rm -rf $(DIST_DIR) completions man

## ── Help ─────────────────────────────────────────────────────────────────────

help:
	@echo ""
	@echo "Nimbi build targets:"
	@echo ""
	@echo "  make build         Build binary for current OS/arch"
	@echo "  make build-all     Cross-compile for all release targets"
	@echo "  make test          Run tests with race detector and coverage"
	@echo "  make bench         Run benchmark suite"
	@echo "  make lint          Run golangci-lint"
	@echo "  make fmt           Format source with gofmt + goimports"
	@echo "  make install       Install binary to $(INSTALL_DIR)"
	@echo "  make release       Full goreleaser release"
	@echo "  make snapshot      Goreleaser snapshot (no publish)"
	@echo "  make completions   Generate shell completions"
	@echo "  make man           Generate man pages"
	@echo "  make clean         Remove all build artifacts"
	@echo ""
	@echo "  VERSION=$(VERSION)"
	@echo ""

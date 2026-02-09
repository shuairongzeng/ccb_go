# ──────────────────────────────────────────────────────────────
# CCB (Claude Code Bridge) Makefile
# ──────────────────────────────────────────────────────────────

# Variables
BINARY      := ccb
MODULE      := github.com/anthropics/claude_code_bridge
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS     := -s -w -X main.version=$(VERSION)
GOFLAGS     := -trimpath
GO          := go
GOLINT      := golangci-lint

# Platform detection
ifeq ($(OS),Windows_NT)
  BINARY_EXT := .exe
  RM_CMD     := del /f /q
else
  BINARY_EXT :=
  RM_CMD     := rm -f
endif

OUTPUT := $(BINARY)$(BINARY_EXT)

.PHONY: all build test lint fmt vet clean install snapshot release help

## ── Default ──────────────────────────────────────────────────
all: lint test build  ## Run lint, test, and build

## ── Build ────────────────────────────────────────────────────
build:  ## Build the binary
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(OUTPUT) ./cmd/ccb/

install:  ## Install to $GOPATH/bin
	$(GO) install $(GOFLAGS) -ldflags "$(LDFLAGS)" ./cmd/ccb/

## ── Quality ──────────────────────────────────────────────────
test:  ## Run tests with race detector
	$(GO) test -v -race -count=1 -coverprofile=coverage.out ./...

test-short:  ## Run tests (short mode, no race detector)
	$(GO) test -short -count=1 ./...

lint:  ## Run golangci-lint
	$(GOLINT) run --timeout=5m ./...

fmt:  ## Format code
	$(GO) fmt ./...
	goimports -w .

vet:  ## Run go vet
	$(GO) vet ./...

## ── Coverage ─────────────────────────────────────────────────
cover: test  ## Generate HTML coverage report
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## ── Cross-compile ────────────────────────────────────────────
build-all:  ## Build for all platforms
	@mkdir -p dist
	GOOS=linux   GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/ccb-linux-amd64       ./cmd/ccb/
	GOOS=linux   GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/ccb-linux-arm64       ./cmd/ccb/
	GOOS=darwin  GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/ccb-darwin-amd64      ./cmd/ccb/
	GOOS=darwin  GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/ccb-darwin-arm64      ./cmd/ccb/
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/ccb-windows-amd64.exe ./cmd/ccb/
	GOOS=windows GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/ccb-windows-arm64.exe ./cmd/ccb/

## ── Release ──────────────────────────────────────────────────
snapshot:  ## GoReleaser snapshot (no publish)
	goreleaser release --snapshot --clean

release:  ## GoReleaser full release (requires GITHUB_TOKEN)
	goreleaser release --clean

## ── Tidy ─────────────────────────────────────────────────────
tidy:  ## Tidy and verify dependencies
	$(GO) mod tidy
	$(GO) mod verify

## ── Clean ────────────────────────────────────────────────────
clean:  ## Remove build artifacts
	$(RM_CMD) $(OUTPUT) 2>/dev/null || true
	$(RM_CMD) coverage.out coverage.html 2>/dev/null || true
	rm -rf dist/ 2>/dev/null || true

## ── Help ─────────────────────────────────────────────────────
help:  ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

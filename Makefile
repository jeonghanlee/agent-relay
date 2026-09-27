SHELL := /bin/bash

VERSION ?= $(shell git -C $(CURDIR) describe --tags --always --dirty 2>/dev/null || echo "0.1.0-dev")
GIT_COMMIT ?= $(shell git -C $(CURDIR) rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

PKG_VERSION := github.com/jeonghanlee/agent-relay/internal/version
LDFLAGS := -s -w -X $(PKG_VERSION).Version=$(VERSION) -X $(PKG_VERSION).GitCommit=$(GIT_COMMIT) -X $(PKG_VERSION).BuildDate=$(BUILD_DATE)

BIN_DIR := bin
TARGET := $(BIN_DIR)/agent-relay

.PHONY: all build test test-mock clean vendor fmt lint

all: build

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

build: $(BIN_DIR)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(TARGET) ./cmd/agent-relay

test:
	go test -race -v ./...

test-mock:
	@if [ -d "pkg/mock" ]; then \
		go test -race -v ./pkg/mock/...; \
	else \
		echo "test-mock: pkg/mock not yet created (scheduled in milestone M4)"; \
	fi

clean:
	rm -rf $(BIN_DIR)

vendor:
	go mod tidy
	go mod vendor

fmt:
	gofmt -s -w .

lint:
	test -z "$$(gofmt -s -l .)"

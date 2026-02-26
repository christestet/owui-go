# Makefile
.PHONY: build test lint clean run

# Variables
BINARY_NAME=owui
BUILD_DIR=bin
GO=go
GOFLAGS=-v

# Build the application
build:
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/owui

# Run tests
test:
	$(GO) test -v -race -coverprofile=coverage.out ./...

# Run the application
run:
	$(GO) run ./cmd/owui

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

# Install dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Format code
fmt:
	$(GO) fmt ./...

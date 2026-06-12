# Makefile
.PHONY: check build test fmt vet lint vuln clean run typecheck deps docs-cli docs-readme docs-readme-check docs docs-deps docs-site docs-dev

# Variables
BINARY_NAME=owui
BUILD_DIR=bin
BIN_DIR=$(CURDIR)/bin
GO=go
GOFLAGS=-v
DOCS_DIR=docs
DOCS_CLI_DIR=$(DOCS_DIR)/src/content/docs/reference/cli
GOLANGCI_LINT_VERSION=v2.12.2
GOLANGCI_LINT=$(BIN_DIR)/golangci-lint
GOVULNCHECK_VERSION=v1.3.0
GOVULNCHECK=$(BIN_DIR)/govulncheck

check: fmt vet lint test vuln docs-readme-check

# Build the application
build:
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/owui

# Run tests
test:
	$(GO) test -v -race -tags='!interactive' -coverprofile=coverage.out ./...

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
	@out=$$(gofmt -l . | grep -v '^bin/' || true); \
	if [ -n "$$out" ]; then echo "gofmt needed for:"; echo "$$out"; exit 1; fi

vet:
	$(GO) vet ./...

$(GOLANGCI_LINT):
	GOBIN=$(BIN_DIR) $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run

$(GOVULNCHECK):
	GOBIN=$(BIN_DIR) $(GO) install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

vuln: $(GOVULNCHECK)
	$(GOVULNCHECK) ./...

typecheck:
	$(GO) tool type-check ./...

# Generate the CLI reference into the Starlight docs content directory
docs-cli:
	$(GO) run ./internal/tools/docgen -out ./$(DOCS_CLI_DIR) -format markdown -frontmatter

docs-readme:
	$(GO) run ./internal/tools/readmegen -readme ./README.md

docs-readme-check: docs-readme
	git diff --exit-code -- README.md

# Install the Astro/Starlight docs site dependencies
docs-deps:
	cd $(DOCS_DIR) && npm ci

# Build the static docs site (generates the CLI reference first)
docs-site: docs-cli
	cd $(DOCS_DIR) && npm run build

# Run the docs site locally with live reload
docs-dev: docs-cli
	cd $(DOCS_DIR) && npm run dev

docs: docs-cli docs-readme

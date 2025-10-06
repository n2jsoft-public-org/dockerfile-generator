# Makefile for dockerfile-gen development workflow
#
# Common targets:
#   make lint         Run golangci-lint (auto-installs pinned version locally into ./bin)
#   make test         Run tests with race detector and coverage
#   make coverage     Show coverage summary
#   make build        Build binary into ./bin/
#   make snapshot     GoReleaser snapshot build (no publish)
#   make release      Full GoReleaser release (requires tag + GH token)
#   make docker       Build local docker image (multi-stage, single arch)
#   make tidy         Ensure go.mod/go.sum tidy
#   make ci           Run lint + test (similar to CI pipeline)
#   make clean        Remove build artifacts

SHELL := /bin/bash

PROJECT        := dockerfile-gen
MODULE         := github.com/n2jsoft-public-org/dotnet-dockerfile-generator
BIN_DIR        := bin
DIST_DIR       := dist
GOLANGCI_LINT_VERSION ?= 2.5.0
GOLANGCI_LINT := $(BIN_DIR)/golangci-lint-$(GOLANGCI_LINT_VERSION)
GO             ?= go
PKGS           := $(shell $(GO) list ./...)
VERSION        ?= dev
COMMIT         := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE           := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS        := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)
COVER_PROFILE  := coverage.out
COVER_MODE     := atomic
COVERAGE_MIN   ?= 60

.PHONY: all help lint test coverage build snapshot release docker tidy ci clean deps

all: build

help:
	@grep -E '^#|^[a-zA-Z_-]+:' Makefile | sed -e 's/:.*//' -e 's/^# //' | awk 'BEGIN{print "Available targets:"} /^[^#]/ {print "  " $$0}'

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

# Install golangci-lint locally in ./bin (pinned version)
$(GOLANGCI_LINT): | $(BIN_DIR)
	@echo "Installing golangci-lint v$(GOLANGCI_LINT_VERSION)...";
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BIN_DIR) v$(GOLANGCI_LINT_VERSION) && mv $(BIN_DIR)/golangci-lint $(GOLANGCI_LINT)
	@$(GOLANGCI_LINT) version

lint: $(GOLANGCI_LINT)
	@echo "Running golangci-lint...";
	@$(GOLANGCI_LINT) run --timeout=5m

# Run tests with race + coverage
test: tidy
	@echo "Running tests...";
	@$(GO) test -race -coverprofile=$(COVER_PROFILE) -covermode=$(COVER_MODE) ./...

coverage: test
	@echo "Coverage summary:";
	@$(GO) tool cover -func=$(COVER_PROFILE) | tail -n 1
	@PCT=$$( $(GO) tool cover -func=$(COVER_PROFILE) | grep total: | awk '{print $$3}' | tr -d '%' ); \
	if awk -v c="$$PCT" -v m="$(COVERAGE_MIN)" 'BEGIN{exit (c+0 >= m+0)?0:1}'; then \
	  echo "Coverage OK ($$PCT% >= $(COVERAGE_MIN)%)"; \
	else \
	  echo "Coverage below threshold: $$PCT% < $(COVERAGE_MIN)%" >&2; exit 1; fi

# Build binary
build: tidy | $(BIN_DIR)
	@echo "Building $(PROJECT)...";
	@CGO_ENABLED=0 $(GO) build -trimpath -ldflags='$(LDFLAGS)' -o $(BIN_DIR)/$(PROJECT) .
	@echo "Built $(BIN_DIR)/$(PROJECT)"

# GoReleaser snapshot (no publish)
snapshot: tidy
	@command -v goreleaser >/dev/null || (echo "goreleaser not installed" >&2; exit 1)
	@goreleaser release --snapshot --skip=publish --skip=announce --clean

# GoReleaser full release (expects a tag & GITHUB_TOKEN)
release: tidy
	@command -v goreleaser >/dev/null || (echo "goreleaser not installed" >&2; exit 1)
	@goreleaser release --clean

# Local docker build (single arch)
docker: build
	@echo "Building docker image locally (tag: $(PROJECT):dev)...";
	docker build -t $(PROJECT):dev --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg DATE=$(DATE) -f Dockerfile .

# Ensure go.mod/sum are tidy
tidy:
	@$(GO) mod tidy
	@git diff --quiet go.mod go.sum || (echo 'go.mod/go.sum not tidy (run make tidy and commit changes)' >&2)

# Aggregate dev workflow
ci: lint test

clean:
	@rm -rf $(BIN_DIR) $(DIST_DIR) $(COVER_PROFILE)
	@echo "Cleaned build artifacts."

deps:
	@$(GO) mod download

# End of Makefile


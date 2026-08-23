.PHONY: all build run run-noauth test fmt fmt-check vet clean docker docker-run verify-repro help

BINARY := meow-server
DOCKER_IMAGE := meow-server
PORT := 8080

# --- Reproducible build configuration ---------------------------------------
# All overridable via `make <target> VAR=value` so `make run GOOS=darwin GOARCH=arm64`
# still works on non-linux hosts. The reproducible artifact is pinned to linux/amd64
# to match the Docker image; on non-linux dev hosts use `make docker-run`.
GOOS        ?= linux
GOARCH      ?= amd64
GOAMD64     ?= v1
CGO_ENABLED ?= 0
GOTOOLCHAIN ?= go1.24.4
SOURCE_DATE_EPOCH ?= 0

# Flags applied to every reproducible `go build`. Mirrored exactly in the Dockerfile.
LDFLAGS        := -s -w
GO_BUILD_FLAGS := -trimpath -buildvcs=false -ldflags='$(LDFLAGS)'
REPRO_ENV      := CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
                  GOAMD64=$(GOAMD64) GOTOOLCHAIN=$(GOTOOLCHAIN) SOURCE_DATE_EPOCH=$(SOURCE_DATE_EPOCH)

all: build

build: ## Build the meow-server binary (reproducible, static linux/amd64)
	$(REPRO_ENV) go build $(GO_BUILD_FLAGS) -o $(BINARY) .

run: build ## Build and run the server (env vars pass through)
	./$(BINARY)

run-noauth: build ## Run with auth disabled (AUTH_ENABLED=false)
	AUTH_ENABLED=false ./$(BINARY)

test: ## Run the Go tests (host-default toolchain/GOOS)
	go test ./...

fmt: ## Format all Go files in place
	gofmt -w .

fmt-check: ## Check formatting without writing (fails if unformatted)
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Files needing gofmt:"; gofmt -l .; exit 1; \
	else echo "gofmt: OK"; fi

vet: ## Run go vet (host-default toolchain/GOOS)
	go vet ./...

clean: ## Remove the built binary and repro artifacts
	rm -f $(BINARY) $(BINARY).repro1 $(BINARY).repro2

verify-repro: ## Build twice in isolated GOCACHEs and assert byte-identical output
	@set -e; \
	tmp1=$$(mktemp -d); tmp2=$$(mktemp -d); \
	trap 'rm -rf "$$tmp1" "$$tmp2"' EXIT; \
	$(REPRO_ENV) GOCACHE="$$tmp1" go build $(GO_BUILD_FLAGS) -o $(BINARY).repro1 .; \
	$(REPRO_ENV) GOCACHE="$$tmp2" go build $(GO_BUILD_FLAGS) -o $(BINARY).repro2 .; \
	sha1=$$(sha256sum $(BINARY).repro1 | awk '{print $$1}'); \
	sha2=$$(sha256sum $(BINARY).repro2 | awk '{print $$1}'); \
	if [ "$$sha1" = "$$sha2" ]; then \
		echo "reproducible: OK ($$sha1)"; rm -f $(BINARY).repro1 $(BINARY).repro2; \
	else \
		echo "NOT reproducible — byte mismatch:"; \
		echo "  build1: $$sha1"; echo "  build2: $$sha2"; \
		cmp $(BINARY).repro1 $(BINARY).repro2 || true; \
		exit 1; \
	fi

docker: ## Build the Docker image (reproducible linux/amd64, BuildKit)
	DOCKER_BUILDKIT=1 docker build --platform=linux/amd64 -t $(DOCKER_IMAGE) .

docker-run: ## Run the server in Docker (maps host:container $(PORT))
	docker run --rm -p $(PORT):$(PORT) $(DOCKER_IMAGE)

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "meow-server — make <target>\n\n"} /^[a-zA-Z0-9_-]+:.*## / { sub(/^[ \t]+/, "", $$2); printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

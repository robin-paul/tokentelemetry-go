VERSION ?= 1.0.0
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS = -s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT)

.PHONY: all build-frontend build-backend build test clean

all: build

build-frontend:
	@echo "==> Building Astro Static Frontend..."
	cd frontend && ( [ -d node_modules ] || npm ci ) && npm run build

build-backend:
	@echo "==> Compiling Static Go Binary..."
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/tokentelemetry ./cmd/tokentelemetry

build: build-frontend build-backend
	@echo "==> Build complete: bin/tokentelemetry"

test:
	go test -v -race ./...

clean:
	rm -rf bin/ internal/web/dist frontend/dist

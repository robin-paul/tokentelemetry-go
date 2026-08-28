VERSION ?= 2.0.0
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS = -s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT)

# Ensure inherited GOROOT does not cause mismatch with the active go binary
unexport GOROOT

.PHONY: all build-frontend build-server build-cli build-all build test test-ui test-ui-smoke test-ui-visual test-ui-headed docker-build kill clean

all: build-all

build-frontend:
	@echo "==> Building Astro Static Frontend..."
	cd frontend && ( [ -d node_modules ] || npm ci ) && npm run build

build-server: build-frontend
	@echo "==> Compiling Hub Server Binary (bin/tt-server)..."
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/tt-server ./cmd/tt-server

build-cli:
	@echo "==> Compiling Collector CLI Binary (bin/tt)..."
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/tt ./cmd/tt

build-all: build-server build-cli
	@echo "==> Successfully built bin/tt and bin/tt-server"

build: build-all

test:
	go test -v -race ./...

test-ui: build-server
	@echo "==> Running Playwright End-to-End Suite..."
	cd test/playwright && ( [ -d node_modules ] || npm ci ) && npx playwright test

test-ui-smoke: build-server
	@echo "==> Running Playwright Smoke Tests..."
	cd test/playwright && ( [ -d node_modules ] || npm ci ) && npx playwright test --grep @smoke

test-ui-visual: build-server
	@echo "==> Running Dual-Server Playwright Visual Regression Diff Suite..."
	cd test/playwright && ( [ -d node_modules ] || npm ci ) && npx playwright test --project=visual

test-ui-headed: build-server
	@echo "==> Running Playwright Tests in Headed Mode..."
	cd test/playwright && ( [ -d node_modules ] || npm ci ) && npx playwright test --headed

docker-build:
	docker build -t tokentelemetry-hub:$(VERSION) -f deploy/Dockerfile .

kill:
	@echo "==> Terminating any running TokenTelemetry processes..."
	@-killall tt-server tokentelemetry tt 2>/dev/null || true
	@-pkill -f "tt-server|tokentelemetry|bin/tt" 2>/dev/null || true

clean:
	rm -rf bin/ internal/web/dist frontend/dist



.PHONY: build dev snapshot clean install-frontend frontend backend

# Default target
all: build

# Install frontend dependencies
install-frontend:
	pnpm install --dir website

# Build frontend only
frontend: install-frontend
	pnpm build --dir website

# Generate and build the Go binary
build: install-frontend
	go generate ./...
	go build -o nginx-config-viewer

# Development server (watches nginx config and serves on :8080)
dev: build
	./nginx-config-viewer -addr :8080 -path /etc/nginx/nginx.conf

# Build backend without frontend (for development)
backend:
	go build -o nginx-config-viewer

# Local snapshot build (without publishing)
snapshot: install-frontend
	goreleaser build --snapshot --clean

# Clean build artifacts
clean:
	rm -f nginx-config-viewer nginx-config-viewer-*
	rm -rf dist/
	rm -rf website/build/

# Build for specific platforms
build-linux-amd64: install-frontend
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o nginx-config-viewer-linux-amd64

build-linux-arm64: install-frontend
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o nginx-config-viewer-linux-arm64

build-darwin-amd64: install-frontend
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o nginx-config-viewer-darwin-amd64

build-darwin-arm64: install-frontend
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o nginx-config-viewer-darwin-arm64

# Run tests (placeholder for future tests)
test:
	go test ./...

# Format code
fmt:
	go fmt ./...
	cd website && pnpm run prettier --write "src/**/*.{ts,tsx}"

# Lint code
lint:
	go vet ./...
	cd website && pnpm run lint

help:
	@echo "Available targets:"
	@echo "  build                 Build the application (frontend + backend)"
	@echo "  dev                   Run development server on :8080"
	@echo "  frontend              Build frontend only"
	@echo "  backend               Build backend only (no frontend rebuild)"
	@echo "  snapshot              Create local snapshot builds for all platforms"
	@echo "  install-frontend      Install frontend dependencies"
	@echo "  clean                 Clean build artifacts"
	@echo "  build-linux-amd64     Build Linux AMD64 binary"
	@echo "  build-linux-arm64     Build Linux ARM64 binary"
	@echo "  build-darwin-amd64    Build macOS AMD64 binary"
	@echo "  build-darwin-arm64    Build macOS ARM64 binary"
	@echo "  test                  Run tests"
	@echo "  fmt                   Format code"
	@echo "  lint                  Lint code"
	@echo "  help                  Show this help message"
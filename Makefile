.PHONY: build install clean test test-unit test-e2e test-e2e-bash test-e2e-go lint build-all dev

# Binary name
BINARY=gcp-emulator
VERSION?=dev

# Build the CLI
build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) ./cmd/gcp-emulator

# Install to $GOPATH/bin
install:
	go install -ldflags "-X main.version=$(VERSION)" ./cmd/gcp-emulator

# Clean build artifacts
clean:
	rm -f $(BINARY)
	rm -f bin/gcp-emulator-e2e*

# Run all tests
test: test-unit test-e2e

# Run unit tests
test-unit:
	go test -v ./internal/...

# Run e2e tests (all)
test-e2e: test-e2e-bash test-e2e-go

# Run bash integration tests
test-e2e-bash:
	@echo "Running bash integration tests..."
	./test/e2e/integration.sh

# Run Go e2e tests
test-e2e-go:
	@echo "Running Go e2e tests..."
	go test -v -timeout 10m ./test/e2e/...

# Run linter
lint:
	go vet ./...
	gofmt -l .

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY)_linux_amd64 ./cmd/gcp-emulator
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY)_darwin_amd64 ./cmd/gcp-emulator
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY)_darwin_arm64 ./cmd/gcp-emulator

# Development: build and run
dev: build
	./$(BINARY)

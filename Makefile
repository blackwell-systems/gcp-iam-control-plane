.PHONY: build install clean test

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

# Run tests
test:
	go test ./...

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

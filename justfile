# Aurora Project Commands

# Default recipe to show help
default:
    @just --list

# Download dependencies
dep:
    go mod download

# Run linter (errcheck disabled: most issues are safe patterns like db.Close in cleanup)
lint:
    golangci-lint run --disable=errcheck

# Run tests
test:
    go test ./...

# Run tests with coverage
test-coverage:
    go test ./... -coverprofile=coverage.out

# Code format and vet
check:
    gofmt -l -s -w .
    goimports -l -w .
    go vet ./...

# Build all platforms
build: check test
    CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build -o aurora-darwin-arm64 cmd/aurora
    CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -o aurora-darwin-amd64 cmd/aurora
    CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o aurora-linux-amd64 cmd/aurora
    CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build -o aurora-linux-arm64 cmd/aurora
    CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -o aurora-windows.exe cmd/aurora

# Build for current platform
build-current:
    go build -o aurora cmd/aurora

# Run the application
run:
    ./aurora-linux-amd64 start

# Build Docker image
image:
    docker build -t pplmx/aurora .

# Start services with docker compose
start:
    docker compose up -d

# Stop services
stop:
    docker compose down

# Restart services
restart:
    docker compose restart

# Development: build image and start
dev: image start

# Production: build image and start
prod: image start

# Clean up
clean:
    go clean
    docker compose down
    rm -f aurora-*
    rm -f coverage.out

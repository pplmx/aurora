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

# Build-time version info derived once, shared by every build recipe. Injects
# the real git ref (tag if reachable, else commit-ish) and build time into
# cmd.Version/cmd.BuildTime so `aurora version` reports what the binary is
# instead of the 0.0.1 placeholder (TASK-206).
ldflags := `printf '%s' "-s -w -X github.com/pplmx/aurora/cmd/aurora/cmd.Version=$(git describe --tags --always --dirty 2>/dev/null || git rev-parse --short HEAD) -X github.com/pplmx/aurora/cmd/aurora/cmd.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"`

# Build all platforms
build: check test
    CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build -trimpath -ldflags="{{ldflags}}" -o aurora-darwin-arm64 ./cmd/aurora
    CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -trimpath -ldflags="{{ldflags}}" -o aurora-darwin-amd64 ./cmd/aurora
    CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -trimpath -ldflags="{{ldflags}}" -o aurora-linux-amd64 ./cmd/aurora
    CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build -trimpath -ldflags="{{ldflags}}" -o aurora-linux-arm64 ./cmd/aurora
    CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -trimpath -ldflags="{{ldflags}}" -o aurora-windows.exe ./cmd/aurora

# Build for current platform
build-current:
    go build -trimpath -ldflags="{{ldflags}}" -o aurora ./cmd/aurora

# Run the application locally (builds for the current platform then runs a sample)
run: build-current
    ./aurora lottery create -p "A,B,C" -s "seed" -c 3

# Build Docker image (version/build-time carried in as build args so the
# image's `aurora version` reports the real source ref, TASK-206)
image:
    docker build --build-arg VERSION="$(git describe --tags --always --dirty 2>/dev/null || git rev-parse --short HEAD)" --build-arg BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)" -t pplmx/aurora .

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
    rm -f aurora-* ./aurora
    rm -f coverage.out

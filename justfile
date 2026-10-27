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
    CGO_ENABLED=0 go test ./...

# Run tests with coverage
test-coverage:
    CGO_ENABLED=0 go test ./... -coverprofile=coverage.out

# Code format and vet. The desktop is a nested module with its own formatter
# gate (`just desktop-check`); root gofmt/goimports must not walk into it, or
# the two recipes disagree about formatting map literals.
check:
    gofmt -l -s -w $(fd -e go -E desktop .)
    goimports -l -w $(fd -e go -E desktop .)
    CGO_ENABLED=0 go vet ./...

# Build-time version info derived once, shared by every build recipe. Injects
# the real git ref (tag if reachable, else commit-ish) and build time into
# cmd.Version/cmd.BuildTime so `aurora version` reports what the binary is
# instead of the 0.0.1 placeholder (TASK-206).
ldflags := `printf '%s' "-s -w -X github.com/pplmx/aurora/cmd/aurora/cmd.Version=$(git describe --tags --always --dirty 2>/dev/null || git rev-parse --short HEAD) -X github.com/pplmx/aurora/cmd/aurora/cmd.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"`

# Same refs injected into cmd/api's own package-main vars (the api recipe
# below); kept separate because the -X paths differ per binary.
api-ldflags := `printf '%s' "-s -w -X main.Version=$(git describe --tags --always --dirty 2>/dev/null || git rev-parse --short HEAD) -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"`

# Same refs for the MyGo desktop binary. It is built from desktop/ (a nested
# module), so this is the last ldflags var; it mirrors the two above.
desktop-ldflags := `printf '%s' "-s -w -X main.Version=$(git describe --tags --always --dirty 2>/dev/null || git rev-parse --short HEAD) -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"`

# Build all platforms
build: check test
    CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build -trimpath -ldflags="{{ldflags}}" -o aurora-darwin-arm64 ./cmd/aurora
    CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -trimpath -ldflags="{{ldflags}}" -o aurora-darwin-amd64 ./cmd/aurora
    CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -trimpath -ldflags="{{ldflags}}" -o aurora-linux-amd64 ./cmd/aurora
    CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build -trimpath -ldflags="{{ldflags}}" -o aurora-linux-arm64 ./cmd/aurora
    CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -trimpath -ldflags="{{ldflags}}" -o aurora-windows.exe ./cmd/aurora

# Build for current platform
build-current:
    CGO_ENABLED=0 go build -trimpath -ldflags="{{ldflags}}" -o aurora ./cmd/aurora

# Build the standalone REST API + Web server binary (cmd/api). Injects the
# same git-derived version/build-time refs into cmd/api's own Version/BuildTime
# vars so `aurora-api --version` reports the real source ref, mirroring the
# CLI's ldflags above (TASK-267, ISS-263).
api:
    CGO_ENABLED=0 go build -trimpath -ldflags="{{api-ldflags}}" -o aurora-api ./cmd/api

# The desktop shell is a nested Go module: its MyGo toolchain stays out of
# the root build graph (that is why this is a separate `cd desktop` build).
# Build the MyGo desktop shell (Go 1.27; storage is pure Go)
desktop:
    cd desktop && CGO_ENABLED=0 go build -trimpath -ldflags="{{desktop-ldflags}}" -o ../aurora-desktop ./cmd/aurora-desktop

# Check for a newer MyGo release before desktop work (the module tracks it closely).
desktop-deps:
    cd desktop && go list -m -u github.com/egoist/mygo

# Update MyGo to the latest release, then let Go reconcile the nested module.
desktop-update-mygo:
    cd desktop && go get github.com/egoist/mygo@latest
    cd desktop && go mod tidy

# Run the desktop shell (build first, then window)
desktop-run: desktop
    ./aurora-desktop

# Test the desktop module. `just test` only covers the core module, so the
# shell's own tests (embedded UI, app-scheme write path, export/import) need
# their own entry point: a broken window layer must not pass CI silently just
# because its module is nested.
desktop-test:
    cd desktop && CGO_ENABLED=0 go test -count=1 ./...

# Every mandatory desktop gate that works in a cgo-free Go sandbox. Kept as a
# recipe so a local run and CI cannot drift: `just test`/`just check` at the
# root do not look inside a nested module at all, so without one of these a
# broken window layer passes every gate.
desktop-check:
    cd desktop && out=$(gofmt -l .) && test -z "$out" && echo "gofmt: clean" || { echo "gofmt would change: $out"; exit 1; }
    cd desktop && CGO_ENABLED=0 go build ./...
    cd desktop && CGO_ENABLED=0 go vet ./...
    cd desktop && CGO_ENABLED=0 go test -count=1 ./...

# The race detector needs cgo even when every production dependency is pure
# Go. Keep it as an explicit opt-in so `desktop-check` remains usable in a
# sandbox with no C compiler while CI still runs the stronger gate.
desktop-race:
    cd desktop && CGO_ENABLED=1 go test -race -count=1 ./...

# Package the desktop app with `go tool mygo build` (desktop/mygo.config.ts):
# a native executable, a desktop entry that registers aurora:// links, a
# Debian package and an installable tarball under dist/desktop/<os>-<arch>/.
# The mygo CLI is a module tool in desktop/go.mod, so its version is pinned
# like any other dependency. Needs Go 1.27; storage is pure Go, so packaging
# works without a native C toolchain.
desktop-package *ARGS:
    cd desktop && CGO_ENABLED=0 go tool mygo build {{ARGS}}

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

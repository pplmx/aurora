# Aurora Project Commands

# Default recipe to show help
default:
    @just --list

# Download dependencies
dep:
    go mod download

# Run the linter with errcheck disabled: most findings are safe patterns like
# db.Close in cleanup.

# Run the linter (errcheck disabled)
lint:
    golangci-lint run --disable=errcheck

# Run tests
test:
    CGO_ENABLED=0 go test ./...

# Run tests with the race detector (needs cgo; the desktop has its own)
race:
    CGO_ENABLED=1 go test -race -count=1 ./...

# Enforce the documented coverage floor on the module domain and app packages
# (the composition root under internal/app/container is wiring, not a module).

# Check the domain and app coverage floor
coverage-floor:
    CGO_ENABLED=0 go test -coverprofile=coverage-floor.out ./internal/domain/... ./internal/app/...
    CGO_ENABLED=0 go run ./tools/coverfloor -profile coverage-floor.out -floor 80

# Run tests with coverage
test-coverage:
    CGO_ENABLED=0 go test ./... -coverprofile=coverage.out

# Code format and vet. The desktop is a nested module with its own formatter
# gate (`just desktop-check`); root gofmt/goimports must not walk into it, or
# the two recipes disagree about formatting map literals.

# Format the core module and run vet
check:
    gofmt -l -s -w $(fd -e go -E desktop .)
    goimports -l -w $(fd -e go -E desktop .)
    CGO_ENABLED=0 go vet ./...

# Regenerate the Traditional Chinese catalogs from the Simplified ones with
# OpenCC's dictionaries (STPhrases.txt and STCharacters.txt from BYVoid/OpenCC;
# pass the directory holding them). A wording fix goes to the Simplified table,
# then this regenerates the Traditional one; `-check` reports a stale catalog.

# Regenerate the Traditional Chinese catalogs
gen-zh-hant dict_dir:
    CGO_ENABLED=0 go run ./tools/genzhhant -dict-dir {{dict_dir}}

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

# Build the standalone REST API binary (cmd/api). Injects the
# same git-derived version/build-time refs into cmd/api's own Version/BuildTime
# vars so `aurora-api --version` reports the real source ref, mirroring the
# CLI's ldflags above (TASK-267, ISS-263).

# Build the standalone REST API binary
api:
    CGO_ENABLED=0 go build -trimpath -ldflags="{{api-ldflags}}" -o aurora-api ./cmd/api

# The desktop shell is a nested Go module: its MyGo toolchain stays out of the
# root build graph (that is why this is a separate `cd desktop` build). The Go
# 1.27 toolchain compiles it; storage is pure Go.

# Build the MyGo desktop shell
desktop:
    cd desktop && CGO_ENABLED=0 go build -trimpath -ldflags="{{desktop-ldflags}}" -o ../aurora-desktop ./cmd/aurora-desktop

# Check for a newer MyGo release before desktop work.

# Check for a newer MyGo release
desktop-deps:
    cd desktop && go list -m -u github.com/egoist/mygo

# Update MyGo to the latest release, then let Go reconcile the nested module.

# Update MyGo to the latest release
desktop-update-mygo:
    cd desktop && go get github.com/egoist/mygo@latest
    cd desktop && go mod tidy

# Run the desktop shell (build first, then window)
desktop-run: desktop
    ./aurora-desktop

# Test the desktop module. `just test` only covers the core module, so the
# shell's own tests (native widget rendering, launch intents, export/import)
# need their own entry point: a broken window layer must not pass CI silently
# just because its module is nested.

# Test the desktop module
desktop-test:
    cd desktop && CGO_ENABLED=0 go test -count=1 ./...

# Every mandatory desktop gate that works in a cgo-free Go sandbox. Kept as a
# recipe so a local run and CI cannot drift: `just test`/`just check` at the
# root do not look inside a nested module at all, so without one of these a
# broken window layer passes every gate.

# Run every cgo-free desktop gate
desktop-check:
    cd desktop && out=$(gofmt -l .) && test -z "$out" && echo "gofmt: clean" || { echo "gofmt would change: $out"; exit 1; }
    cd desktop && CGO_ENABLED=0 go build ./...
    cd desktop && CGO_ENABLED=0 go vet ./...
    cd desktop && golangci-lint run --config ../.golangci.yml --disable=errcheck ./...
    cd desktop && CGO_ENABLED=0 go test -count=1 ./...

# The views carry the behaviour and the adapters carry the platform glue; a
# floor keeps a refactor from deleting the tests that cover them. The command
# binaries are wiring and stay out of the check.

# Check the desktop coverage floor
desktop-coverage-floor:
    cd desktop && CGO_ENABLED=0 go test -coverprofile=coverage-desktop.out ./internal/...
    cd desktop && CGO_ENABLED=0 go run ../tools/coverfloor/main.go -profile coverage-desktop.out -prefixes github.com/pplmx/aurora/desktop/internal/ -exclude "" -floor 70

# The race detector needs cgo even when every production dependency is pure
# Go. Keep it as an explicit opt-in so `desktop-check` remains usable in a
# sandbox with no C compiler while CI still runs the stronger gate.

# Run the desktop tests under the race detector
desktop-race:
    cd desktop && CGO_ENABLED=1 go test -race -count=1 ./...

# Open the real window and capture each page as a PNG through MyGo's automation
# plugin. The offscreen tester covers rendering, but only this path proves a
# built binary draws what the tests expect; it complements the packaging job's
# Xvfb smoke. Pass page ids to limit the run: `just desktop-shot token nft`.
# Needs a display; Linux without one starts an Xvfb.

# Capture each page in the real window as a PNG
desktop-shot *PAGES:
    desktop/scripts/e2e-window.sh {{PAGES}}

# Package the desktop app with `go tool mygo build` (desktop/mygo.config.ts):
# a native executable, a desktop entry that registers aurora:// links, a
# Debian package and an installable tarball under dist/desktop/<os>-<arch>/.
# The mygo CLI is a module tool in desktop/go.mod, so its version is pinned
# like any other dependency. Needs Go 1.27; storage is pure Go, so packaging
# works without a native C toolchain.

# Package the desktop app with go tool mygo
desktop-package *ARGS:
    cd desktop && CGO_ENABLED=0 go tool mygo build {{ARGS}}

# Build for this platform and run a sample draw
run: build-current
    ./aurora lottery create -p "A,B,C" -s "seed" -c 3

# Build Docker image (version/build-time carried in as build args so the
# image's `aurora version` reports the real source ref, TASK-206).

# Build the Docker image
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

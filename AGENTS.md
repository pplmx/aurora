# AGENTS.md - Aurora Project Guide

## Build & Test Commands

```bash
make test         # Run all tests
make lint         # Run golangci-lint (requires golangci-lint installed)
make check        # Run gofmt, goimports, go vet
make build        # Runs check + test, then builds for darwin/linux/windows
make run          # Run the binary locally (./aurora-linux start)
```

## Dev Workflow

```bash
make dev          # Build Docker image and restart containers
make start        # Start containers: docker compose up -d
make stop         # Stop containers: docker compose down
```

## Project Structure

- **Entry point**: `cmd/aurora/main.go` → `cmd/aurora/cmd/root.go`
- **Core logic**: `internal/` (blockchain, voting, logger, utils)
- **Config**: `config/aurora.x.yml` (Viper loads from `$HOME` or `./config/`)

## Known Issues

- **Dockerfile bug**: Line 18 references `src/main.go` but the actual entry point is `cmd/aurora/main.go`. Build will fail unless fixed.

## Configuration

- Config file format: YAML
- Default config name: `aurora.x.yml` (not `.yaml`)
- Config lookup order: CLI flag → `$HOME/aurora.x.yml` → `./config/aurora.x.yml`
- Default log level: `info`
- Default log path: `./logs/`

## Dependencies

- Go 1.26.0
- Cobra (CLI framework)
- Viper (config)
- Zerolog (logging)
# AGENTS.md - Aurora Project Guide

## Build & Test Commands

```bash
make test         # Run all tests (unit + E2E)
make lint         # Run golangci-lint (requires golangci-lint installed)
make check        # Run gofmt, goimports, go vet
make build        # Runs check + test, then builds for darwin/linux/windows
make run          # Run locally: ./aurora-linux lottery create -p "A,B,C" -s "seed" -c 3
```

## Dev Workflow

```bash
make dev          # Build Docker image and restart containers
make start        # Start containers: docker compose up -d
make stop         # Stop containers: docker compose down
```

## Project Structure

- **Entry point**: `cmd/aurora/main.go` → `cmd/aurora/cmd/root.go`
- **Lottery module**: `internal/lottery/` (VRF, TUI, lottery logic)
- **Core logic**: `internal/` (blockchain, voting, logger, utils)
- **Tests**: `internal/lottery/*_test.go` (unit), `test/lottery_e2e_test.go` (E2E)
- **Config**: `config/aurora.x.toml` (Viper loads from `$HOME` or `./config/`)

## Configuration

- Config file format: TOML
- Default config name: `aurora.x.toml`
- Config lookup order: CLI flag → `$HOME/aurora.x.toml` → `./config/aurora.x.toml`
- Default log level: `info`
- Default log path: `./logs/`

## Dependencies

- Go 1.26+
- Cobra (CLI framework)
- Viper (config)
- Zerolog (logging)
- filippo.io/edwards25519 (VRF)
- charmbracelet/bubbletea (TUI)
- charmbracelet/lipgloss (styling)

## Lottery Commands

```bash
# CLI
./aurora lottery create -p "A,B,C,D" -s "seed" -c 3   # Create lottery
./aurora lottery history                               # View history
./aurora lottery tui                                   # TUI interface
```

## Testing

```bash
go test ./internal/lottery/ -v    # Unit tests
go test ./test/ -v                # E2E tests
go test ./...                     # All tests
```

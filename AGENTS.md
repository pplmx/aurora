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
- **Lottery module**: `internal/domain/lottery/`, `internal/ui/lottery/`
- **Voting module**: `internal/domain/voting/`
- **NFT module**: `internal/domain/nft/`, `internal/app/nft/`, `internal/ui/nft/`
- **Token (FT) module**: `internal/domain/token/`, `internal/app/token/`, `internal/ui/token/`
- **Oracle module**: `internal/domain/oracle/`, `internal/app/oracle/`, `internal/ui/oracle/`
- **Core logic**: `internal/domain/blockchain/`, `internal/i18n/`, `internal/utils/`
- **Tests**: `e2e/*_test.go` (E2E), `internal/*/ *_test.go` (unit)
- **Config**: `config/aurora.toml` (Viper loads from `$HOME` or `./config/`)

## Configuration

- Config file format: TOML
- Default config name: `aurora.toml`
- Config lookup order: CLI flag → `$HOME/aurora.toml` → `./config/aurora.toml`
- Default log level: `info`
- Default log path: `./logs/`

## Dependencies

- Go 1.26+
- Cobra (CLI framework)
- Viper (config)
- Zerolog (logging)
- filippo.io/edwards25519 (VRF, Ed25519 signing)
- charmbracelet/bubbletea (TUI)
- charmbracelet/lipgloss (styling)

## Module Commands

### Lottery (VRF-based)

```bash
./aurora lottery create -p "A,B,C,D" -s "seed" -c 3   # Create lottery
./aurora lottery history                               # View history
./aurora lottery tui                                   # TUI interface
```

### Voting (Ed25519 signatures)

```bash
./aurora voting create -t "Proposal" -o "owner_key"   # Create vote
./aurora voting vote -k "signing_key"                 # Cast vote
./aurora voting tui                                   # TUI interface
```

### NFT (Ed25519-signed NFTs)

```bash
./aurora nft mint -n "MyNFT" -d "Description" -c "creator_key"
./aurora nft transfer --nft <id> --to <address> -k "private_key"
./aurora nft get --id <nft_id>
./aurora nft list --owner <pubkey>
./aurora nft tui                                       # TUI interface
```

### Token (Fungible Token)

```bash
./aurora token create -n "MyToken" -s "SYMBOL" --supply 1000000
./aurora token mint --to <address> --amount 100 -k "private_key"
./aurora token transfer --to <address> --amount 50 -k "private_key"
./aurora token balance --owner <address>
./aurora token history
./aurora token tui                                       # TUI interface
```

### Oracle (Data Oracle)

```bash
./aurora oracle sources                                 # List data sources
./aurora oracle fetch --source <id>                    # Fetch data
./aurora oracle query --source <id> --limit 10         # Query history
./aurora oracle tui                                       # TUI interface
```

## Testing

```bash
go test ./internal/domain/... -v    # Domain layer tests
go test ./internal/app/... -v       # Application layer tests
go test ./e2e/ -v                   # E2E tests
go test ./...                       # All tests
go test ./... -cover               # With coverage
```

## Test Coverage

| Module | Domain | App |
|--------|--------|-----|
| Lottery | 74.0% | 87.1% |
| Voting | 87.5% | 81.4% |
| NFT | 72.7% | 88.1% |
| Token | 70.7% | 40.7% |
| Oracle | 76.1% | 65.8% |
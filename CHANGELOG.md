# Changelog

All notable changes to this project will be documented in this file.

## [1.1.0] - 2026-04-06

### Refactored - DDD Architecture

Complete migration from monolithic modules to **Domain-Driven Design (DDD)** architecture:

- **Domain Layer** (`internal/domain/`): Entities, Services, Repository interfaces
  - `blockchain/` - Block, BlockChain, InitBlockChain()
  - `lottery/` - LotteryRecord, VRF Service, Repository
  - `voting/` - Vote, Voter, Candidate, Session
  - `nft/` - NFT, Operation, Service
  - `oracle/` - OracleData, DataSource

- **Infrastructure Layer** (`internal/infra/`): Storage implementations
  - `sqlite/` - SQLite repository implementations
  - `http/` - HTTP client for Oracle

- **Application Layer** (`internal/app/`): Use Cases
  - `lottery/` - CreateLotteryUseCase
  - `voting/` - CastVoteUseCase, RegisterVoterUseCase
  - `nft/` - MintNFTUseCase, TransferNFTUseCase
  - `oracle/` - FetchDataUseCase

- **Presentation Layer** (`internal/ui/`): TUI interfaces
  - `lottery/`, `nft/`, `oracle/`

### Dependencies Updated

- Bubble Tea v1 → v2 (`charm.land/bubbletea/v2`)
- Bubbles v1 → v2 (`charm.land/bubbles/v2`)
- Lipgloss v1 → v2 (`charm.land/lipgloss/v2`)
- GitHub Actions updated to latest versions

### Added

- **Domain Tests**: Unit tests for domain entities
- **SECURITY.md**: Security policy and vulnerability reporting
- **.goreleaser.yaml**: GoReleaser configuration
- **Pre-commit hooks**: Go format/lint hooks

### Fixed

- Docker build cache issues
- Go version in Dockerfile (1.26)
- golangci-lint configuration
- errcheck disabled in CI (matches local lint)

## [1.0.0] - 2024-04-06

### Added

- **VRF Lottery System**: Transparent lottery using Verifiable Random Function with blockchain storage
- **Voting System**: Ed25519 signed voting with real-time counting and blockchain persistence
- **Oracle Module**: HTTP API data fetching with predefined templates (BTC/ETH prices)
- **NFT System**: Mint, transfer, and burn NFTs with Ed25519 signatures
- **Blockchain Storage**: SQLite persistence with Write-through caching
- **CLI Commands**: Full command-line interface for all modules
- **TUI Interface**: Interactive terminal UI using Bubble Tea
- **E2E Tests**: Comprehensive end-to-end tests for all modules
- **CI/CD**: GitHub Actions workflows (test, build, docker, release)
- **Pre-commit Hooks**: Go format/lint hooks

### Changed

- Replaced Makefile with justfile
- Split CI workflows into separate files

### Features

- Cross-platform builds (darwin/linux/windows, amd64/arm64)
- Docker multi-platform images (amd64/arm64)
- Auto-release on tag
- Codecov integration

## [0.0.0] - 2024-04-05

### Initial Release

- Basic lottery system with VRF
- In-memory blockchain
- CLI and TUI interfaces
# AGENTS.md - Aurora Project Guide

## Build & Test Commands

> The project uses **just** (see `justfile`), not make. There is no Makefile.

```bash
just test         # Run all tests (unit + E2E)
just lint         # Run golangci-lint (requires golangci-lint installed)
just check        # Run gofmt, goimports, go vet
just build        # Runs check + test, then builds for darwin/linux/windows
just build-current  # Build for the current platform (./aurora)
just run          # Run locally
```

## Dev Workflow

```bash
just dev          # Build Docker image and restart containers
just start        # Start containers: docker compose up -d
just stop         # Stop containers: docker compose down
```

## Development Notes

- **不使用 worktree**: 个人开发直接在 main 分支进行，无需创建 worktree
- 使用 worktree 会导致最后合并时产生不必要的冲突

## Git Workflow

- **禁止使用 `git commit --amend`**：每次提交应该是独立的、完整的。如果需要修改，使用新的提交
- **提交粒度：完整但不零碎，也不过大**。
  - 不要每个小改动单独一个 commit —— 一组小的相关改动（同一 task / 同一逻辑单元）应合并进**一个完整 commit**。
  - 反过来也不要一个 commit 塞进太大范围的东西 —— 以"一个自洽的逻辑单元"为单位（一次 bug 修复、一个 feature、一次同主题重构），参考 commit message 里的 task/issue 编号（TASK-x / ISS-x）来判断边界。
  - 实践中一个里程碑通常由 3–4 个功能级 commit 加 1 个文档收尾 commit 组成，而非几十个碎片。
- 提交前确保所有 pre-commit hooks 通过
- 如果 hooks 自动修改了文件，需要将这些修改添加到提交中

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

## Go Style

- 写/改 Go 代码前调用 `use-modern-go` 技能（Skill tool 名称
  `modern-go-guidelines:use-modern-go`），按返回的 guidelines 写现代惯用 Go
  （内置 `min`/`max`、`slices`/`maps` 泛型 helper、`errors.Is`/`As`、`any` 等）。
  新代码遵循 guideline；跳过某条 guideline 时用它的 `explain` 子命令确认理由。

## Module Commands

### Lottery (VRF-based)

```bash
./aurora lottery create -p "A,B,C,D" -s "seed" -c 3   # Create lottery
./aurora lottery history                               # View history
./aurora lottery verify <id-or-height>                 # Verify a draw's VRF proof & winners
./aurora lottery export file.json                      # Export records
./aurora lottery import file.json                      # Import records
./aurora lottery stats                                 # Show lottery stats
./aurora lottery tui                                   # TUI interface
```

### Voting (Ed25519 signatures)

```bash
./aurora voting candidate add -n "Name" -p "Party"     # Register a candidate
./aurora voting candidate list                          # List candidates
./aurora voting voter register -n "Name"               # Register a voter (prints keypair)
./aurora voting session create -t "Title" -c <cand-id> -c <cand2> --start-time <t> --end-time <t>
./aurora voting session list                           # List sessions
./aurora voting session start -s <session-id>          # Start a session
./aurora voting vote -v <voter-pk> -c <candidate-id> -s <session-id> -k <priv-key>
./aurora voting results -s <session-id>                # Show results
./aurora voting session end -s <session-id>            # End a session
```

### NFT (Ed25519-signed NFTs)

```bash
./aurora nft mint -n "MyNFT" -d "Description" -c "creator_key"
./aurora nft transfer --nft <id> --from <owner> --to <address> -k "private_key"
./aurora nft get --nft <nft_id>
./aurora nft list --owner <pubkey>
./aurora nft burn --nft <id> --owner <owner> -k "private_key" --confirm   # 需 --confirm
./aurora nft history --nft <id>
./aurora nft tui                                       # TUI interface
```

### Token (Fungible Token)

```bash
./aurora token create -n "MyToken" -s "SYMBOL" --supply 1000000   # prints owner keypair
./aurora token info -t <token-id>
./aurora token mint -t <token-id> --to <address> --amount 100 -k "private_key"
./aurora token transfer -t <token-id> --from <addr> --to <address> --amount 50 -k "private_key"
./aurora token balance -t <token-id> --owner <address>
./aurora token approve -t <token-id> --owner <addr> --spender <addr> --amount 100 -k "private_key"
./aurora token allowance -t <token-id> --owner <addr> --spender <addr>
./aurora token transfer-from -t <token-id> -o <owner> --to <addr> -a <amount> -s <spender> -k <spender_key>
./aurora token burn -t <token-id> --from <addr> --amount 10 -k "private_key" --confirm   # 需 --confirm
./aurora token history -t <token-id> --owner <address>
./aurora token tui                                       # TUI interface
```

### Oracle (Data Oracle)

```bash
./aurora oracle source list                             # List data sources
./aurora oracle source add -n "Name" -u <url>           # Add a source (-m/--method, -p/--path, --interval optional)
./aurora oracle fetch -i <source-id>                    # Fetch data (-s legacy)
./aurora oracle data -i <source-id> --limit 10          # Query history
./aurora oracle latest -i <source-id>                   # Latest data point
./aurora oracle template list                           # Built-in templates
./aurora oracle template add -t <template>              # Add a template source
./aurora oracle tui                                     # TUI interface
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

> 覆盖率的单一事实源在 README「测试覆盖率」一节（`go test -cover` 实测，带
> `as of` 日期）。这里不再复制数字，避免两处过期副本相互漂移。

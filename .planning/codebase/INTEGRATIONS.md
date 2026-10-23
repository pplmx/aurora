# External Integrations

**Analysis Date:** 2026-10-23

## APIs & External Services

**Oracle Data Sources:**

- Sources are persisted in SQLite at runtime through `aurora oracle source add`,
  the REST API, or the Oracle TUI. `config/aurora.toml` intentionally does not
  read an `[[oracle.sources]]` table.
- Built-in templates and operator-defined HTTP sources can target arbitrary
  public HTTP(S) endpoints. SSRF checks run both when a source is added and
  immediately before each fetch.

**Custom HTTP Fetcher:**

- Location: `internal/infra/http/fetcher.go`
- Features:
    - Rate limiting (configurable requests per window)
    - Request timeout (default: 10s)
    - Security headers (User-Agent, Accept, Content-Type, X-Request-ID)
    - Path extraction from JSON responses
- Rate Limit Config:

  ```toml
  [http.rateLimit]
  requests = 10
  window = "1m"
  ```

## Data Storage

**SQLite Database:**

- Driver: `modernc.org/sqlite v1.61.0` (pure Go)
- Path: `./data/aurora.db` (configurable)
- Mode: WAL (Write-Ahead Logging)
- Schema migrations: `migrations/*.sql`

**Tables:**

| Table             | Purpose                         |
| ----------------- | ------------------------------- |
| `blocks`          | Blockchain block storage        |
| `lottery_records` | VRF-based lottery results       |
| `votes`           | Ed25519-signed votes            |
| `voters`          | Voter registration              |
| `candidates`      | Voting candidates               |
| `voting_sessions` | Voting periods                  |
| `nfts`            | NFT ownership records           |
| `nft_operations`  | NFT transfer/mint history       |
| `tokens`          | Fungible token definitions      |
| `accounts`        | Token balances                  |
| `allowances`      | Token allowances (ERC-20 style) |
| `data_sources`    | Oracle source configurations    |
| `oracle_data`     | Fetched data records            |
| `events`          | Durable audit event log         |
| `pending_events`  | Audit-delivery outbox           |
| `nonces`          | Token replay protection         |
| `token_transfers` | Legacy table; no production I/O |

**File Storage:**

- Local filesystem only
- Data directory: `./data/`
- Log directory: `./logs/`

## Authentication & Identity

**Cryptographic Signatures:**

- Ed25519 (via `filippo.io/edwards25519`)
- Used for:
    - NFT minting and transfers
    - Voting (vote authentication)
    - Lottery VRF proofs

**No External Auth Provider:**

- Self-contained identity system
- Key pairs generated/managed locally
- Signatures verified on-chain (simulated)

## Monitoring & Observability

**Logging:**

- Framework: `rs/zerolog v1.35.1`
- Format: Structured JSON to stdout and file
- Default level: `info`
- Config path: `./logs/`

**Test Coverage Tracking:**

- codecov/codecov-action@v6 (GitHub Actions)
- Coverage reports: `coverage.out`

**Health and Metrics Endpoints:**

- `/healthz`, `/readyz`, and `/health` are served by `aurora-api`.
- `/metrics` exposes bounded request counters and latency histograms;
  `/metrics/oracle` exposes per-source fetch health.
- The Docker image contains only the CLI and therefore has no HTTP health
  endpoint or Docker `HEALTHCHECK`.

## CI/CD & Deployment

**GitHub Actions:**

- **ci.yml** - Run tests on push/PR to main
    - Go 1.27.2
    - Cgo-free test gate plus an explicit race-detector gate
    - Codecov upload
    - Linting with golangci-lint v2.15.0

- **build.yml** - Cross-platform binary builds
    - Platforms: darwin, linux, windows
    - Architectures: amd64, arm64
    - Artifacts uploaded for 5 days
    - Go build metadata must report `CGO_ENABLED=0` and the expected target

- **desktop-build.yml** - Native desktop packaging and signed release matrix
- **release.yml** - Release automation

**Docker:**

- Base image: `golang:1.27-alpine` (builder)
- Runtime: `gcr.io/distroless/static:nonroot`
- Image: `pplmx/aurora:latest` (Docker Hub)
- Published via GitHub Actions
- Builder uses `CGO_ENABLED=0`; SQLite requires no native C library

**Kubernetes:**

- Manifest: `k8s.yml`

## Environment Configuration

**Config File:** `config/aurora.toml`

**Required Settings:**

```toml
[server]
host = "0.0.0.0"
port = 8080

[log]
level = "info"
path = "./logs/"

[db]
path = "./data/aurora.db"

[http]
timeout = "10s"

[http.rateLimit]
requests = 10
window = "1m"
```

## Web Interface

**Static Files:**

- Location: `web/`
- Files: HTML, CSS, JavaScript
- Embedded into the CLI/API/desktop binaries and served by the same Chi router
- The desktop shell serves the same router over `aurora://` without a TCP
  listener

**HTTP Server:**

- Framework: `go-chi/chi/v5`
- Location: `internal/api/`
- Routes: `internal/api/router.go`
- Handlers: `internal/api/handler/`

---

*Integration audit: 2026-10-23*

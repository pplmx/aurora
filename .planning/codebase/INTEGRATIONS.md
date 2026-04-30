# External Integrations

**Analysis Date:** 2026-04-30

## APIs & External Services

**Oracle Data Sources:**
- **CoinGecko API** - Cryptocurrency price feeds
  - Endpoint: `https://api.coingecko.com/api/v3/simple/price`
  - Sources configured: Bitcoin (BTC) and Ethereum (ETH) prices in USD
  - Usage: `internal/infra/http/fetcher.go` - Rate-limited HTTP client with configurable requests/window
  - Config: `config/aurora.toml` section `[[oracle.sources]]`

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
- Driver: `github.com/mattn/go-sqlite3 v1.14.41`
- Path: `./data/aurora.db` (configurable)
- Mode: WAL (Write-Ahead Logging)
- Schema migrations: `migrations/*.sql`

**Tables:**
| Table | Purpose |
|-------|---------|
| `blocks` | Blockchain block storage |
| `lottery_records` | VRF-based lottery results |
| `votes` | Ed25519-signed votes |
| `voters` | Voter registration |
| `candidates` | Voting candidates |
| `voting_sessions` | Voting periods |
| `nfts` | NFT ownership records |
| `nft_operations` | NFT transfer/mint history |
| `tokens` | Fungible token definitions |
| `accounts` | Token balances |
| `allowances` | Token allowances (ERC-20 style) |
| `data_sources` | Oracle source configurations |
| `oracle_data` | Fetched data records |

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
- Framework: `rs/zerolog v1.35.0`
- Format: Structured JSON to stdout and file
- Default level: `info`
- Config path: `./logs/`

**Test Coverage Tracking:**
- codecov/codecov-action@v6 (GitHub Actions)
- Coverage reports: `coverage.out`

**Health Check:**
- Docker healthcheck: `./aurora lottery stats`
- Interval: 30s

## CI/CD & Deployment

**GitHub Actions:**
- **ci.yml** - Run tests on push/PR to main
  - Go 1.26
  - Race detector enabled
  - Codecov upload
  - Linting (golangci-lint)

- **build.yml** - Cross-platform binary builds
  - Platforms: darwin, linux, windows
  - Architectures: amd64, arm64
  - Artifacts uploaded for 5 days

- **docker.yml** - Docker image builds
- **release.yml** - Release automation

**Docker:**
- Base image: `golang:1.26-alpine` (builder)
- Runtime: `gcr.io/distroless/static:nonroot`
- Image: `pplmx/aurora:latest` (Docker Hub)
- Published via GitHub Actions

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
type = "sqlite"
path = "./data/aurora.db"

[http]
timeout = "10s"

[http.rateLimit]
requests = 10
window = "1m"
```

**Oracle Source Example:**
```toml
[[oracle.sources]]
id = "btc-price"
name = "Bitcoin Price"
url = "https://api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=usd"
type = "price"
method = "GET"
path = "bitcoin.usd"
interval = 60
enabled = true
```

## Web Interface

**Static Files:**
- Location: `web/`
- Files: HTML, CSS, JavaScript
- Served by embedded file server or external nginx

**HTTP Server:**
- Framework: `go-chi/chi/v5`
- Location: `internal/api/`
- Routes: `internal/api/router.go`
- Handlers: `internal/api/handler/`

---

*Integration audit: 2026-04-30*
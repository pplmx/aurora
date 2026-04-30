# Plan 08-04: Implement Graceful Shutdown

**Phase:** 8 - Operations & Health
**Requirements:** OPS-03
**Status:** Planned

## Tasks

### 1. Update main.go shutdown handling
**File:** `cmd/api/main.go`

Replace the shutdown section (lines 49-52):

```go
import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pplmx/aurora/internal/api"
	"github.com/pplmx/aurora/internal/config"
	"github.com/pplmx/aurora/internal/logger"
)

const shutdownTimeout = 15 * time.Second

func main() {
	// ... existing setup code (lines 15-43) ...

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info().Str("signal", sig.String()).Msg("Shutting down server...")

	// Graceful shutdown: stop accepting new connections, wait for in-flight
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("Server shutdown error")
		// Fall back to force close if graceful shutdown fails
		if err := server.Close(); err != nil {
			logger.Error().Err(err).Msg("Server force close error")
		}
	}

	logger.Info().Msg("Server stopped")
}
```

## Files to Modify

| File | Action |
|------|--------|
| `cmd/api/main.go` | Add context import, update shutdown logic |

## Success Criteria

- [ ] Server listens for SIGINT and SIGTERM signals
- [ ] On shutdown signal, server stops accepting new connections
- [ ] In-flight requests complete before server exits (up to 15 seconds)
- [ ] If graceful shutdown times out, server force closes
- [ ] Log messages show shutdown progress and any errors

## Testing

```bash
# Start server
./aurora-api &

# Send SIGTERM
kill -TERM $PID

# Expected logs:
# level=info msg="Shutting down server..." signal=TERMINATED
# level=info msg="Server stopped"

# For in-flight request test:
# 1. Make a long-running API request
# 2. Send SIGTERM during request
# 3. Request should complete (not fail with connection reset)
```

## Key Differences

| Approach | New Connections | In-Flight Requests | Timeout |
|----------|-----------------|-------------------|---------|
| `server.Close()` (old) | Immediately rejected | Force closed | None |
| `server.Shutdown(ctx)` (new) | Stop accepted | Wait for completion | 15 seconds |
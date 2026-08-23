package sqlite

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"

	"github.com/pplmx/aurora/internal/domain/events"
	"github.com/pplmx/aurora/internal/domain/token"
	infraevents "github.com/pplmx/aurora/internal/infra/events"
	"github.com/stretchr/testify/require"
)

// TestGetTransferEventsByOwner_InterleavedTypesRegressions locks the v1.61 fix:
// pagination must apply over TRANSFER events only, not every event in the token
// aggregate. Previously GetTransferEventsByOwner passed limit to
// GetByAggregate, whose SQL LIMIT counted mints/burns/transfers together, so
// interleaved mint events crowded out transfers and a limit=5 request returned
// fewer than 5 transfers (TASK-074, ISS-066). This uses the real SQLite store.
func TestGetTransferEventsByOwner_InterleavedTypesRegression(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "token_transfer_paging_*.db")
	require.NoError(t, err)
	_ = tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	store, err := infraevents.NewSQLiteEventStore(tmpFile.Name())
	require.NoError(t, err)
	defer store.Close()

	owner := []byte("owner")
	recip := []byte("recip")
	ownerB := base64.StdEncoding.EncodeToString(owner)
	recipB := base64.StdEncoding.EncodeToString(recip)

	// Interleave a transfer and a mint per iteration so the aggregate's raw
	// event stream alternates types (the pre-fix bug).
	for i := 0; i < 20; i++ {
		tp, _ := json.Marshal(map[string]interface{}{"from": ownerB, "to": recipB, "amount": "10", "nonce": i, "sig": "s"})
		require.NoError(t, store.Save(events.NewBaseEvent("token.transfer", "T", tp)))
		mp, _ := json.Marshal(map[string]interface{}{"to": ownerB, "amount": "5"})
		require.NoError(t, store.Save(events.NewBaseEvent("token.mint", "T", mp)))
	}

	reader := NewTokenEventReader(store)

	// limit=5 must now return exactly 5 transfers despite the 20 mint events.
	got, err := reader.GetTransferEventsByOwner(token.TokenID("T"), owner, 5, 0)
	require.NoError(t, err)
	require.Len(t, got, 5, "transfer history page must be composed of transfers only")

	// Offset pagination must step through the transfer stream, not the mixed
	// aggregate stream.
	got2, err := reader.GetTransferEventsByOwner(token.TokenID("T"), owner, 5, 5)
	require.NoError(t, err)
	require.Len(t, got2, 5)
	require.Equal(t, "10", got2[0].Amount().String(), "second page should still be transfer events")
}

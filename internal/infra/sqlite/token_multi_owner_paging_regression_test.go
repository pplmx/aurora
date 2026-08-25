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

// TestGetTransferEventsByOwner_MultiOwnerPagination locks the TASK-093 fix:
// history pagination must apply over ONE owner's transfers, not over every
// transfer in the token aggregate. Previously GetTransferEventsByOwner ran
// SQL LIMIT/OFFSET over all token.transfer events and then filtered the page
// for the requested owner in memory, so with >=2 transferring owners each page
// under-filled (a limit=10 request for an owner with 10+ transfers returned 5)
// and offset advanced through the whole token's stream, skipping the owner's
// own transfers. Now owner is part of the SQL predicate and each page is a
// full, owner-scoped page (ISS-086).
func TestGetTransferEventsByOwner_MultiOwnerPagination(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "token_multi_owner_paging_*.db")
	require.NoError(t, err)
	_ = tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	store, err := infraevents.NewSQLiteEventStore(tmpFile.Name())
	require.NoError(t, err)
	defer store.Close()

	alice := []byte("alice")
	bob := []byte("bob")
	recip := []byte("recip")
	aliceB := base64.StdEncoding.EncodeToString(alice)
	bobB := base64.StdEncoding.EncodeToString(bob)
	recipB := base64.StdEncoding.EncodeToString(recip)

	// 20 transfers alternating owners: alice, bob, alice, bob, ...
	for i := 0; i < 20; i++ {
		from := aliceB
		if i%2 == 1 {
			from = bobB
		}
		tp, _ := json.Marshal(map[string]interface{}{"from": from, "to": recipB, "amount": "10", "nonce": i, "sig": "s"})
		require.NoError(t, store.Save(events.NewBaseEvent("token.transfer", "T", tp)))
	}

	reader := NewTokenEventReader(store)

	// Alice is the "from" of every even transfer (10 total). A limit=10 page
	// for her must return all 10, not the 5 that survive an in-memory filter
	// over a 10-row all-owners SQL page.
	page1, err := reader.GetTransferEventsByOwner(token.TokenID("T"), alice, 10, 0)
	require.NoError(t, err)
	require.Len(t, page1, 10, "owner scoped page must be full even on a multi-owner token")

	all := map[string]bool{}
	for _, e := range page1 {
		require.False(t, all[e.ID()], "page 1 must not contain duplicate events")
		all[e.ID()] = true
		require.Equal(t, "10", e.Amount().String())
	}

	// Offset paging must step through Alice's own stream. With 10 Alice
	// transfers total, offset=10 has nothing left.
	page2, err := reader.GetTransferEventsByOwner(token.TokenID("T"), alice, 10, 10)
	require.NoError(t, err)
	require.Empty(t, page2, "offset must be relative to the owner's stream, not the token's")

	// Bob must similarly get a full page from his own interleaved stream.
	bobPage, err := reader.GetTransferEventsByOwner(token.TokenID("T"), bob, 10, 0)
	require.NoError(t, err)
	require.Len(t, bobPage, 10, "each owner must page its own full history")
}

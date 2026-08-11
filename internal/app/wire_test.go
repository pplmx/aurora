package app

import (
	"crypto/ed25519"
	"testing"

	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/stretchr/testify/require"
)

// TestWire_TokenRoundTrip drives a full token lifecycle through the composed
// App (separate events.db / nonces.db / tokens.db files, distinct connection
// pools, sync bus -> audit handler -> event store) and asserts the ledger plus
// the audit trail stay consistent. This is the integration coverage the
// composition root previously had none of.
func TestWire_TokenRoundTrip(t *testing.T) {
	app, err := Wire(t.TempDir())
	require.NoError(t, err)

	svc := app.TokenService

	ownerPub, ownerPriv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	created, err := svc.CreateToken(&token.CreateTokenRequest{
		Name:        "Wire Token",
		Symbol:      "WIRE",
		TotalSupply: token.NewAmount(1000),
		Owner:       token.PublicKey(ownerPub),
	})
	require.NoError(t, err)
	require.Equal(t, token.TokenID("WIRE"), created.ID())

	// Mint 500 to the owner (mintable by default, owner-authorized).
	_, err = svc.Mint(&token.MintRequest{
		TokenID:    "WIRE",
		To:         token.PublicKey(ownerPub),
		Amount:     token.NewAmount(500),
		PrivateKey: token.PrivateKey(ownerPriv),
	})
	require.NoError(t, err)

	// Transfer 200 from owner to a second address.
	toPub, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	_, err = svc.Transfer(&token.TransferRequest{
		TokenID:    "WIRE",
		From:       token.PublicKey(ownerPub),
		To:         token.PublicKey(toPub),
		Amount:     token.NewAmount(200),
		PrivateKey: token.PrivateKey(ownerPriv),
	})
	require.NoError(t, err)

	// Ledger invariants across the separate DB files.
	info, err := svc.GetTokenInfo("WIRE")
	require.NoError(t, err)
	require.Equal(t, int64(1500), info.TotalSupply().Int64(), "supply = 1000 created + 500 minted")

	ownerBal, err := svc.GetBalance("WIRE", token.PublicKey(ownerPub))
	require.NoError(t, err)
	require.Equal(t, int64(1300), ownerBal.Int64(), "1000 initial + 500 mint - 200 transfer")

	toBal, err := svc.GetBalance("WIRE", token.PublicKey(toPub))
	require.NoError(t, err)
	require.Equal(t, int64(200), toBal.Int64())

	// Audit trail through events.db must contain the transfer.
	hist, err := svc.GetTransferHistory("WIRE", token.PublicKey(ownerPub), 10, 0)
	require.NoError(t, err)
	require.NotEmpty(t, hist, "transfer history must be recorded in the event store")
}

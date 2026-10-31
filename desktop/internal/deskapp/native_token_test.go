package deskapp

import (
	"path/filepath"
	"testing"

	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	infraevents "github.com/pplmx/aurora/internal/infra/events"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/stretchr/testify/require"
)

// tokenServices assembles the adapter over a real SQLite-backed token service,
// the same way the container does. The token paths sign and persist, so a fake
// service would only prove the adapter calls it.
func tokenServices(t *testing.T) (*nativeServices, string, string) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "aurora.db")

	repo, err := sqlite.NewTokenRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	eventStore, err := infraevents.NewSQLiteEventStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = eventStore.Close() })
	eventReader := sqlite.NewTokenEventReader(eventStore)
	eventBus := infraevents.NewSyncEventBus()
	// The transfer stream is read back from the persisted events, so the bus
	// needs the same audit subscription production wires: without it the events
	// are published in memory and never reach the reader.
	eventBus.SubscribeAll(infraevents.NewAuditHandlerWithOutbox(eventStore, eventStore).Handle)
	replay, err := infraevents.NewSQLiteReplayProtection(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = replay.Close() })

	chain, err := blockchain.NewWithDB(repo.GetDB())
	require.NoError(t, err)

	service := token.NewService(repo, sqlite.NewTxManager(repo.GetDB()), eventBus, eventReader, replay, chain)
	services := &nativeServices{tokenService: service}

	ownerPub, ownerPriv, err := services.GenerateKeyPair()
	require.NoError(t, err)
	_, err = services.CreateToken("Gold Reserve", "GLD", "1000000", ownerPub, 8)
	require.NoError(t, err)
	return services, ownerPub, ownerPriv
}

// TestTokenInfoCarriesEveryField pins the read the Inspect card uses. Decimals
// is the field most likely to be dropped: it is an int8 in the model and an int
// in the view row, and the card showed 0 before the conversion was added.
func TestTokenInfoCarriesEveryField(t *testing.T) {
	services, ownerPub, _ := tokenServices(t)

	info, err := services.TokenInfo("GLD")
	require.NoError(t, err)
	require.Equal(t, "GLD", info.ID)
	require.Equal(t, "Gold Reserve", info.Name)
	require.Equal(t, "GLD", info.Symbol)
	require.Equal(t, "1000000", info.TotalSupply)
	require.Equal(t, ownerPub, info.Owner)
	require.Equal(t, 8, info.Decimals)
}

// TestTokenReadsReportAMissingToken keeps the failure path honest: the card must
// show the store's reason rather than an empty record.
func TestTokenReadsReportAMissingToken(t *testing.T) {
	services, _, _ := tokenServices(t)

	_, err := services.TokenInfo("NO-SUCH")
	require.Error(t, err)
}

// TestTokenMintAndBalance walks the first write an operator performs: the
// balance follows the mint. The transfer stream is checked after a transfer,
// because the history holds transfer events only — a mint is its own event type.
func TestTokenMintAndBalance(t *testing.T) {
	services, ownerPub, ownerPriv := tokenServices(t)

	require.NoError(t, services.MintToken("GLD", ownerPub, "250", ownerPriv))

	// Creating a token credits its whole supply to the owner, so the balance
	// after a mint is the supply plus the minted amount.
	balance, err := services.TokenBalance("GLD", ownerPub)
	require.NoError(t, err)
	require.Equal(t, "1000250", balance, "the mint adds to the supply the owner already holds")

	// A transfer, so the stream the History card reads has something in it.
	toPub, _, err := services.GenerateKeyPair()
	require.NoError(t, err)
	require.NoError(t, services.TransferToken("GLD", ownerPub, toPub, "10", ownerPriv))

	rows, err := services.TokenHistory("GLD", "", 0, 0)
	require.NoError(t, err)
	require.Len(t, rows, 1, "the transfer is what the stream holds")
	require.Equal(t, "10", rows[0].Amount)
}

// TestTokenTransferMovesValue covers the write the Transfer tab performs,
// including that the sender's balance drops by the amount the recipient gains.
func TestTokenTransferMovesValue(t *testing.T) {
	services, ownerPub, ownerPriv := tokenServices(t)
	toPub, _, err := services.GenerateKeyPair()
	require.NoError(t, err)
	require.NoError(t, services.MintToken("GLD", ownerPub, "100", ownerPriv))

	require.NoError(t, services.TransferToken("GLD", ownerPub, toPub, "40", ownerPriv))

	sender, err := services.TokenBalance("GLD", ownerPub)
	require.NoError(t, err)
	require.Equal(t, "1000060", sender, "the supply the owner started with, plus the mint, less the transfer")
	recipient, err := services.TokenBalance("GLD", toPub)
	require.NoError(t, err)
	require.Equal(t, "40", recipient)
}

// TestTokenAllowanceRoundTrip covers the read the Allowance card performs: the
// granted amount is what the card shows, and a fresh pair starts at zero.
func TestTokenAllowanceRoundTrip(t *testing.T) {
	services, ownerPub, ownerPriv := tokenServices(t)
	spenderPub, _, err := services.GenerateKeyPair()
	require.NoError(t, err)

	granted, err := services.TokenAllowance("GLD", ownerPub, spenderPub)
	require.NoError(t, err)
	require.Equal(t, "0", granted, "no allowance before it is granted")

	require.NoError(t, services.ApproveToken("GLD", ownerPub, spenderPub, "75", ownerPriv))

	granted, err = services.TokenAllowance("GLD", ownerPub, spenderPub)
	require.NoError(t, err)
	require.Equal(t, "75", granted)
}

// TestTokenBurnReducesSupply covers the destructive write: the burned amount
// leaves the account, which is the fact the confirmation dialog promised.
func TestTokenBurnReducesSupply(t *testing.T) {
	services, ownerPub, ownerPriv := tokenServices(t)
	require.NoError(t, services.MintToken("GLD", ownerPub, "100", ownerPriv))

	require.NoError(t, services.BurnToken("GLD", ownerPub, "30", ownerPriv))

	balance, err := services.TokenBalance("GLD", ownerPub)
	require.NoError(t, err)
	require.Equal(t, "1000070", balance, "the supply plus the mint, less the burned amount")
}

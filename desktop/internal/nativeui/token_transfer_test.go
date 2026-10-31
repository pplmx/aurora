package nativeui

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// transferApp fills the Transfer form and returns the app, so each test only
// describes what it is about.
func transferApp(t *testing.T, err error) (*App, *fakeServices) {
	t.Helper()
	fake := &fakeServices{transferErr: err}
	app := New(fake)
	ui.NewTester(app.View, 1280, 1400).Frame()
	app.ShowPage("Token")
	app.token.tokenID = "GLD"
	app.token.xFrom = "sender-pk"
	app.token.xTo = "recipient-pk"
	app.token.xAmount = "40"
	app.token.xPriv = "sender-priv"
	return app, fake
}

// TestTransferPassesTheFormToTheService pins what the button sends: the four
// fields and the key, trimmed, in the order the use case expects.
func TestTransferPassesTheFormToTheService(t *testing.T) {
	app, fake := transferApp(t, nil)
	app.token.tokenID = "  GLD  "
	app.token.xFrom = " sender-pk "

	app.transferToken()

	require.Equal(t, 1, fake.transferTokenCalls)
	require.Equal(t, [5]string{"GLD", "sender-pk", "recipient-pk", "40", "sender-priv"}, fake.lastTransfer,
		"the service gets trimmed values, and the key it must sign with")
}

// TestSuccessfulTransferClearsTheKey is the security half: a signing key that
// worked is not kept on screen, because the next write needs its own and a
// shared display should not retain the secret.
func TestSuccessfulTransferClearsTheKey(t *testing.T) {
	app, _ := transferApp(t, nil)

	app.transferToken()

	require.Empty(t, app.token.xPriv, "the key is cleared after a successful write")
	require.False(t, app.token.status.isErr, "a completed write is not an error")
	require.Equal(t, "Done.", app.token.status.text, "the operator is told the write landed")
}

// TestFailedTransferKeepsTheKeyForARetry is the other half of the same rule: a
// rejected transfer must leave the form as the operator typed it, or the retry
// costs a re-paste of the key.
func TestFailedTransferKeepsTheKeyForARetry(t *testing.T) {
	app, _ := transferApp(t, errors.New("insufficient balance"))

	app.transferToken()

	require.Equal(t, "sender-priv", app.token.xPriv,
		"a failed transfer keeps the key, so the retry does not need it re-entered")
	require.True(t, app.token.status.isErr)
	require.Contains(t, app.token.status.text, "insufficient balance")
}

// TestTransferRefusesAnIncompleteForm covers the guard: a missing field is a
// refusal with a sentence naming what is needed, not a request with blanks.
func TestTransferRefusesAnIncompleteForm(t *testing.T) {
	app, fake := transferApp(t, nil)
	app.token.xAmount = "   "

	app.transferToken()

	require.Zero(t, fake.transferTokenCalls, "an incomplete form must not reach the service")
	require.True(t, app.token.status.isErr)
	require.Contains(t, app.token.status.text, "Token ID, sender, recipient, amount, and private key are required.")
}

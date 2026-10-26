package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestSessionKeyMemoryIsOptInAndFillsTheForm covers the security boundary:
// nothing is remembered unless the user enabled it, and a remembered key can
// fill the next form without touching the disk.
func TestSessionKeyMemoryIsOptInAndFillsTheForm(t *testing.T) {
	const owner = "owner-public-key"
	fake := &fakeServices{tokenRows: []TokenRow{{ID: "t1", Name: "Gold", Owner: owner}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Token")
	tester.Frame()

	app.token.tokenID = "t1"
	app.token.mintTo = owner
	app.token.mintAmount = "5"
	app.token.mintPriv = "owner-private-key"
	app.mintToken()
	require.Zero(t, app.rememberedKeyCount(), "memory is off by default")

	app.rememberKeys = true
	app.token.mintPriv = "owner-private-key"
	app.mintToken()
	require.Equal(t, 1, app.rememberedKeyCount())
	require.Empty(t, app.token.mintPriv, "the successful write still clears the field")

	app.token.mintPriv = ""
	tester.Frame()
	require.True(t, tester.HasText("Use saved key"))
	require.NoError(t, tester.Click("Use saved key"))
	tester.Frame()
	require.Equal(t, "owner-private-key", app.token.mintPriv)
}

// TestForgetSessionKeysClearsWhatFormsCanReuse keeps the Settings action
// honest: forgetting removes the in-memory entries and the form stops offering
// the shortcut in the next frame.
func TestForgetSessionKeysClearsWhatFormsCanReuse(t *testing.T) {
	app := New(&fakeServices{})
	app.rememberKeys = true
	app.rememberSuccessfulKey("pk", "sk")
	require.Equal(t, 1, app.rememberedKeyCount())

	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Settings")
	tester.Frame()
	require.NoError(t, tester.Click("Forget session keys"))
	tester.Frame()
	require.Zero(t, app.rememberedKeyCount())
	_, ok := app.savedKey("pk")
	require.False(t, ok)
}

// TestSessionKeyCheckboxIsTheOptIn pins the visible control behind the memory
// boundary.
func TestSessionKeyCheckboxIsTheOptIn(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Settings")
	tester.Frame()
	require.False(t, app.rememberKeys)

	require.NoError(t, tester.Click("Remember private keys for this session"))
	tester.Frame()
	require.True(t, app.rememberKeys)

	app.rememberSuccessfulKey("pk", "sk")
	require.Equal(t, 1, app.rememberedKeyCount())
	require.NoError(t, tester.Click("Remember private keys for this session"))
	tester.Frame()
	require.False(t, app.rememberKeys)
	require.Zero(t, app.rememberedKeyCount(), "turning memory off must clear the keys")
}

// TestVoterRegistrationRemembersTheBallotKey covers the one-time key a voter
// registration returns: with memory enabled it is available to later ballots
// without another copy from the key pair card.
func TestVoterRegistrationRemembersTheBallotKey(t *testing.T) {
	app := New(&voterKeyRecorder{fakeServices: &fakeServices{}})
	app.rememberKeys = true
	app.voting.voterName = "Alice"

	app.registerVoter()

	key, ok := app.savedKey("pk-Alice")
	require.True(t, ok)
	require.Equal(t, "sk-Alice", key)
}

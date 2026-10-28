package nativeui

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// testKeyPair returns a real base64 Ed25519 pair: session memory derives the
// public half from the private key, so placeholder strings would be rejected.
func testKeyPair(t *testing.T, seed byte) (public, private string) {
	t.Helper()
	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, ed25519.SeedSize))
	return base64.StdEncoding.EncodeToString(key.Public().(ed25519.PublicKey)),
		base64.StdEncoding.EncodeToString(key)
}

// TestSessionKeyMemoryIsOptInAndFillsTheForm covers the security boundary:
// nothing is remembered unless the user enabled it, and a remembered key can
// fill the next form without touching the disk.
func TestSessionKeyMemoryIsOptInAndFillsTheForm(t *testing.T) {
	owner, ownerPrivate := testKeyPair(t, 1)
	fake := &fakeServices{tokenRows: []TokenRow{{ID: "t1", Name: "Gold", Owner: owner}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Token")
	tester.Frame()

	app.token.tokenID = "t1"
	app.token.mintTo = owner
	app.token.mintAmount = "5"
	app.token.mintPriv = ownerPrivate
	app.mintToken()
	require.Zero(t, app.rememberedKeyCount(), "memory is off by default")

	app.rememberKeys = true
	app.token.mintPriv = ownerPrivate
	app.mintToken()
	require.Equal(t, 1, app.rememberedKeyCount())
	require.Empty(t, app.token.mintPriv, "the successful write still clears the field")

	app.token.mintPriv = ""
	tester.Frame()
	require.True(t, tester.HasText("Use saved key"))
	require.NoError(t, tester.Click("Use saved key"))
	tester.Frame()
	require.Equal(t, ownerPrivate, app.token.mintPriv)
}

// TestForgetSessionKeysClearsWhatFormsCanReuse keeps the Settings action
// honest: the destructive trigger opens a safe confirmation, and only the
// deliberate red action removes the in-memory entries.
func TestForgetSessionKeysClearsWhatFormsCanReuse(t *testing.T) {
	pk, sk := testKeyPair(t, 2)
	app := New(&fakeServices{})
	app.rememberKeys = true
	app.rememberSuccessfulKey(pk, sk)
	require.Equal(t, 1, app.rememberedKeyCount())

	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Settings")
	tester.Frame()
	require.NoError(t, tester.Click("Forget session keys"))
	tester.Frame()
	require.Equal(t, 1, app.rememberedKeyCount(), "opening the dialog must not erase keys")
	require.True(t, tester.HasText("Forget all session keys?"))
	require.True(t, tester.Focused(tr("Cancel")), "the safe action must receive initial focus")

	require.NoError(t, tester.Click("Cancel"))
	tester.Frame()
	require.Equal(t, 1, app.rememberedKeyCount())
	require.True(t, tester.Focused("Forget session keys"),
		"Cancel must return focus to the button that opened the dialog")

	require.NoError(t, tester.Click("Forget session keys"))
	tester.Frame()
	require.NoError(t, tester.Click("Forget keys"))
	tester.Frame()
	require.Zero(t, app.rememberedKeyCount())
	_, ok := app.savedKey(pk)
	require.False(t, ok)
	require.True(t, app.rememberKeys, "forgetting the current keys leaves the opt-in enabled")
}

// TestSessionKeyCheckboxIsTheOptIn pins the visible control behind the memory
// boundary. Turning it off is a two-step action when it would discard keys:
// Cancel preserves both the policy and the keys, while the red action applies
// both changes.
func TestSessionKeyCheckboxIsTheOptIn(t *testing.T) {
	pk, sk := testKeyPair(t, 3)
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Settings")
	tester.Frame()
	require.False(t, app.rememberKeys)

	require.NoError(t, tester.Click("Remember private keys for this session"))
	tester.Frame()
	require.True(t, app.rememberKeys)

	app.rememberSuccessfulKey(pk, sk)
	require.Equal(t, 1, app.rememberedKeyCount())
	require.NoError(t, tester.Click("Remember private keys for this session"))
	tester.Frame()
	require.True(t, app.rememberKeys, "the prompt must not turn memory off before confirmation")
	require.Equal(t, 1, app.rememberedKeyCount(), "the prompt must preserve the keys")
	require.True(t, tester.HasText("Turn off session key memory?"))
	require.True(t, tester.Focused(tr("Cancel")))

	require.NoError(t, tester.Click("Cancel"))
	tester.Frame()
	require.True(t, app.rememberKeys)
	require.Equal(t, 1, app.rememberedKeyCount(), "Cancel must preserve the keys")
	require.True(t, tester.Focused("Remember private keys for this session"),
		"Cancel must return focus to the checkbox that opened the dialog")

	require.NoError(t, tester.Click("Remember private keys for this session"))
	tester.Frame()
	require.NoError(t, tester.Click("Forget keys"))
	tester.Frame()
	require.False(t, app.rememberKeys)
	require.Zero(t, app.rememberedKeyCount(), "turning memory off must clear the keys")
}

// TestTurningOffEmptySessionKeyMemoryDoesNotPrompt keeps the confirmation
// proportional: with nothing to lose, disabling the opt-in stays one click.
func TestTurningOffEmptySessionKeyMemoryDoesNotPrompt(t *testing.T) {
	app := New(&fakeServices{})
	app.rememberKeys = true
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Settings")
	tester.Frame()

	require.NoError(t, tester.Click("Remember private keys for this session"))
	tester.Frame()
	require.False(t, app.rememberKeys)
	require.False(t, app.confirmForgetKeys)
	require.False(t, tester.HasText("Turn off session key memory?"))
	require.Zero(t, app.rememberedKeyCount())
}

// TestVoterRegistrationRemembersTheBallotKey covers the one-time key a voter
// registration returns: with memory enabled it is available to later ballots
// without another copy from the key pair card.
func TestVoterRegistrationRemembersTheBallotKey(t *testing.T) {
	pk, sk := testKeyPair(t, 4)
	app := New(&fixedVoterKeys{fakeServices: &fakeServices{}, public: pk, private: sk})
	app.rememberKeys = true
	app.voting.voterName = "Alice"

	app.registerVoter()

	key, ok := app.savedKey(pk)
	require.True(t, ok)
	require.Equal(t, sk, key)
}

// TestTransferFromCanUseASavedSpenderKey covers the one operation whose signer
// public key is not otherwise part of the form: the optional spender field is
// only a lookup key, and a successful transfer remembers the pair.
func TestTransferFromCanUseASavedSpenderKey(t *testing.T) {
	pk, sk := testKeyPair(t, 5)
	app := New(&fakeServices{tokenRows: []TokenRow{{ID: "t1", Owner: "owner"}}})
	app.rememberKeys = true
	app.token.opTab = 3
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Token")
	tester.Frame()

	app.token.tokenID = "t1"
	tester.Frame()
	app.token.fromOwner, app.token.fromTo, app.token.fromAmount = "owner", "bob", "5"
	app.token.fromSpender, app.token.fromPriv = pk, sk
	app.transferFromToken()
	require.Equal(t, sk, app.savedKeyValue(t, pk))

	app.token.fromPriv = ""
	tester.Frame()
	require.True(t, tester.HasText("Use saved key"))
	require.NoError(t, tester.Click("Use saved key"))
	tester.Frame()
	require.Equal(t, sk, app.token.fromPriv)
}

func (a *App) savedKeyValue(t *testing.T, publicKey string) string {
	t.Helper()
	key, ok := a.savedKey(publicKey)
	require.True(t, ok)
	return key
}

// fixedVoterKeys lets the key-memory test use a real pair without changing the
// fake registration that other tests pin.
type fixedVoterKeys struct {
	*fakeServices
	public, private string
}

func (r *fixedVoterKeys) RegisterVoter(name string) (VoterRow, error) {
	return VoterRow{PublicKey: r.public, Name: name, PrivateKey: r.private}, nil
}

// TestSessionKeyMemoryUsesTheDerivedPublicKey keeps an optional public field
// from mapping a private key to the wrong signer: the pair is stored under the
// key the private half really belongs to, and an explicit mismatch is refused.
func TestSessionKeyMemoryUsesTheDerivedPublicKey(t *testing.T) {
	pk, sk := testKeyPair(t, 6)
	app := New(&fakeServices{})
	app.rememberKeys = true

	app.rememberSuccessfulKey("a-different-public-key", sk)
	require.Zero(t, app.rememberedKeyCount(), "a mismatched pair must not be remembered")

	app.rememberSuccessfulKey("", sk)
	require.Equal(t, sk, app.savedKeyValue(t, pk))
}

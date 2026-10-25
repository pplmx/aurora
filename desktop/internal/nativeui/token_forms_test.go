package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// tokenRecorder is a fakeServices that remembers which account each write
// form sent, so a test can prove the page reads the input the operator filled
// in rather than a leftover from another section.
type tokenRecorder struct {
	*fakeServices
	approveOwner   string
	approveSpender string
	fromOwner      string
	fromTo         string
	allowOwner     string
	allowSpender   string
}

func (r *tokenRecorder) ApproveToken(tokenID, owner, spender, amount, priv string) error {
	r.approveOwner, r.approveSpender = owner, spender
	return nil
}

func (r *tokenRecorder) TransferFromToken(tokenID, owner, to, amount, spenderPriv string) error {
	r.fromOwner, r.fromTo = owner, to
	return nil
}

func (r *tokenRecorder) TokenAllowance(tokenID, owner, spender string) (string, error) {
	r.allowOwner, r.allowSpender = owner, spender
	return "10", nil
}

// TestTokenWriteFormsUseTheirOwnInputs pins that Approve, Transfer From and
// Allowance each read the fields they display. Approve and Allowance used to
// have no owner input at all: they silently sent the Transfer section's "From"
// address, so a spender granted from the Approve form was signed for whichever
// account the operator had typed into a different form — and the section gave
// no way to see or change it. Transfer From borrowed Transfer's recipient the
// same way.
//
// The handlers are driven directly: a click on a section title and a click on
// the button below it are different controls, and the Tester resolves a label
// by text, so it would hit whichever one it found first.
func TestTokenWriteFormsUseTheirOwnInputs(t *testing.T) {
	rec := &tokenRecorder{fakeServices: &fakeServices{}}
	app := New(rec)

	// A leftover Transfer address must never be what another form signs for.
	app.token.tokenID = "t1"
	app.token.xFrom = "transfer-from"
	app.token.xTo = "transfer-to"

	app.token.approveOwner = "approve-owner"
	app.token.approveTo = "approve-spender"
	app.token.approveAmt = "5"
	app.token.approvePriv = "priv"
	app.approveToken()
	require.Equal(t, "approve-owner", rec.approveOwner,
		"Approve must sign as the owner named in the Approve form")
	require.Equal(t, "approve-spender", rec.approveSpender)

	app.token.fromOwner = "allowance-source"
	app.token.fromTo = "from-to"
	app.token.fromAmount = "3"
	app.token.fromPriv = "priv"
	app.transferFromToken()
	require.Equal(t, "allowance-source", rec.fromOwner)
	require.Equal(t, "from-to", rec.fromTo,
		"Transfer From must send the recipient named in its own form")

	app.token.allowanceOwner = "allow-owner"
	app.token.allowanceTo = "allow-spender"
	app.loadAllowance()
	require.Equal(t, "allow-owner", rec.allowOwner,
		"Allowance must read the owner named in the Allowance form")
	require.Equal(t, "allow-spender", rec.allowSpender)
}

// TestTokenFormsRenderTheirOwnerInputs pins the fields the sections show, so a
// form cannot silently lose the account it acts on again.
func TestTokenFormsRenderTheirOwnerInputs(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 4200)
	app.ShowPage("Token")
	tester.Frame()

	texts := tester.Texts()
	for _, want := range []string{
		"Owner (base64 public key)", // Approve, Transfer From and Allowance
		"Spender (base64 public key)",
	} {
		count := 0
		for _, got := range texts {
			if got == want {
				count++
			}
		}
		require.Positive(t, count, "the Token page must show a %q input", want)
	}
}

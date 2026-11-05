package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// clickPrimaryButton finds the primary button labelled label and clicks it.
//
// Tester.Find answers with the first element showing the text, and for a card's
// action the first such element is the whole card — its centre is a form field.
// So the search walks the card's last rows from its left edge, which is where a
// primary button sits, and stops at the first click the predicate accepts.
//
// The predicate is what proves the click landed: a handler that ran leaves an
// observed change, and the caller states what that change is.
func clickPrimaryButton(t *testing.T, tester *ui.Tester, label string, landed func() bool, reset func()) {
	t.Helper()

	card, ok := tester.Find(label)
	require.Truef(t, ok, "no element shows %q", label)

	for offset := card.H - 8; offset > card.H*0.5; offset -= 6 {
		for _, x := range []float32{card.X + 40, card.X + 80, card.X + 120, card.X + 160} {
			reset()
			tester.ClickAt(x, card.Y+offset)
			tester.Frame()
			if landed() {
				return
			}
		}
	}
	require.FailNowf(t, "the button never received the click",
		"clicking %q did not reach its handler; no test proves the control is wired", label)
}

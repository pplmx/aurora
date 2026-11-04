package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestAnUnsafeSourceURLIsRefusedAtItsControl covers where the source form's URL
// refusal belongs. The store applies an SSRF policy — scheme, host shape, and the
// refusal of loopback and cloud-metadata addresses — and its message names no
// field, so the sentence used to arrive in the card's status line above a form of
// seven controls.
//
// The three cases below are the policy's distinct reasons: an address that is not
// a URL at all, a loopback host, and the cloud metadata endpoint. Each must be
// refused before the write, on the URL control.
func TestAnUnsafeSourceURLIsRefusedAtItsControl(t *testing.T) {
	for _, tc := range []struct{ name, url string }{
		{"not a URL", "not-a-url"},
		{"loopback host", "http://localhost:8080/x"},
		{"metadata endpoint", "http://169.254.169.254/latest"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := workbenchFakes()
			app := New(fake)
			tester := ui.NewTester(app.View, 1280, 2600)
			app.ShowPage("Oracle")
			tester.Frame()

			app.oracle.addName, app.oracle.addURL = "feed", tc.url
			app.addOracleSource()
			tester.Frame()

			require.NotEmpty(t, app.oracle.addURLErr, "the refusal belongs on the URL control")
			require.True(t, tester.HasText(app.oracle.addURLErr))
			require.Empty(t, app.oracle.status.text,
				"the status line must not also report a write that never ran")
		})
	}
}

// TestAnAllowedSourceURLCreatesTheSource keeps the check from refusing what the
// policy allows: an ordinary https address reaches the service.
func TestAnAllowedSourceURLCreatesTheSource(t *testing.T) {
	app := New(workbenchFakes())
	tester := ui.NewTester(app.View, 1280, 2600)
	app.ShowPage("Oracle")
	tester.Frame()

	app.oracle.addName, app.oracle.addURL = "feed", "https://example.com/api"
	app.addOracleSource()
	tester.Frame()

	require.Empty(t, app.oracle.addURLErr)
	require.Contains(t, app.oracle.status.text, "Added")
}

// TestTheURLErrorClearsOnTheNextAttempt: retrying with a good address must not
// leave the previous refusal on screen beside a success message. The field's
// Changed handler clears it as the operator types, and a fresh attempt clears it
// as well, so the sentence never outlives the value it described.
func TestTheURLErrorClearsOnTheNextAttempt(t *testing.T) {
	app := New(workbenchFakes())
	tester := ui.NewTester(app.View, 1280, 2600)
	app.ShowPage("Oracle")
	tester.Frame()

	app.oracle.addName, app.oracle.addURL = "feed", "not-a-url"
	app.addOracleSource()
	tester.Frame()
	require.NotEmpty(t, app.oracle.addURLErr)

	app.oracle.addURL = "https://example.com/api"
	app.addOracleSource()
	tester.Frame()
	require.Empty(t, app.oracle.addURLErr,
		"a corrected address must not carry the old refusal")
	require.Contains(t, app.oracle.status.text, "Added")
}

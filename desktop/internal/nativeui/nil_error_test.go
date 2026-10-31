package nativeui

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestANilErrorProducesNoStatusLine covers what a caller gets when a service
// returns no error: an empty line, not a line saying "null" or the string
// "<nil>". Every write handler builds its status from whatever the use case
// returned, so the nil case is reached on every success.
func TestANilErrorProducesNoStatusLine(t *testing.T) {
	line := errStatus(nil)
	require.False(t, line.isErr, "a nil error is not a failure")
	require.Empty(t, line.text, "a nil error carries no message")
	// A zero revision marks "nothing to announce". Consuming one here would
	// make an empty line look like a fresh outcome and announce it.
	require.Zero(t, line.revision, "a nil error must not consume a status revision")
}

// TestANilErrorLocalizesToAnEmptyString pins the other half: the message
// helper returns "" rather than panicking, because a handler that wraps the
// result into a longer sentence still expects a string.
func TestANilErrorLocalizesToAnEmptyString(t *testing.T) {
	require.Empty(t, localizeServiceError(nil))
}

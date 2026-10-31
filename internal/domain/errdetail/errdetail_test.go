package errdetail

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWrapKeepsTheDetailOutOfTheChain pins the contract: callers branch on the
// sentinel, and the detail is printed but never becomes a classification target.
func TestWrapKeepsTheDetailOutOfTheChain(t *testing.T) {
	sentinel := errors.New("sentinel")
	detail := errors.New("parser said no")

	err := Wrap(sentinel, detail)
	require.Error(t, err)
	assert.Equal(t, "sentinel: parser said no", err.Error())
	assert.True(t, errors.Is(err, sentinel), "the sentinel is the contract")
	assert.False(t, errors.Is(err, detail), "the detail must stay out of the chain")
}

package sqlite

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/repoerr"
	"github.com/pplmx/aurora/internal/domain/voting"
)

// TestSentinelsAreTheDomainErrors keeps the compatibility aliases honest:
// application code can classify a repository failure without importing this
// package.
func TestSentinelsAreTheDomainErrors(t *testing.T) {
	require.ErrorIs(t, ErrNotFound, repoerr.ErrNotFound)
	require.ErrorIs(t, ErrAlreadyVoted, voting.ErrAlreadyVoted)
}

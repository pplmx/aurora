package test

import (
	"os"
	"testing"

	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
)

func TestMain(m *testing.M) {
	// Reset blockchain state before all tests
	blockchain.ResetForTest()

	// Run tests
	exitCode := m.Run()

	// Clean after all tests
	blockchain.ResetForTest()

	os.Exit(exitCode)
}

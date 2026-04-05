package test

import (
	"os"
	"testing"

	"github.com/pplmx/aurora/internal/blockchain"
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

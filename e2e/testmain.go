package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"

	"github.com/pplmx/aurora/internal/infra/blockchain"
)

// TestMain keeps this suite's database out of the checkout: the store is
// process-wide and resolves db.path on first open, so without a private
// directory the default ./data would be shared with any other test binary
// running at the same time, and one reset would pull the chain from under the
// other.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "aurora-e2e-*")
	if err != nil {
		panic(err)
	}
	viper.Set("db.path", filepath.Join(dir, "aurora.db"))

	blockchain.ResetForTest()
	code := m.Run()
	blockchain.ResetForTest()

	_ = os.RemoveAll(dir)
	os.Exit(code)
}

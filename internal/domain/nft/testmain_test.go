package nft

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"

	"github.com/pplmx/aurora/internal/infra/blockchain"
)

// TestMain gives this package its own database: the store is process-wide and
// resolves db.path on first open, and the default points at ./data inside the
// checkout. Two test binaries running at once (go test ./... in parallel, or two
// suites at the same time) then share that directory and reset each other's
// chain mid-test.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "aurora-nft-test-*")
	if err != nil {
		panic(err)
	}
	viper.Set("db.path", filepath.Join(dir, "aurora.db"))

	code := m.Run()

	blockchain.ResetForTest()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

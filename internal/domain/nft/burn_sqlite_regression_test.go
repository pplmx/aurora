package nft_test

import (
	"crypto/ed25519"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/infra/sqlite"
)

// stubChain is a tiny BlockWriter so the service under test never touches the
// process-wide blockchain singleton / ./data/aurora.db.
type stubChain struct{ height int64 }

func (c *stubChain) AddBlock(string) (int64, error) {
	c.height++
	return c.height, nil
}

// TestNFTService_Burn_RetainsAuditTrailOverSQLite locks TASK-092 / ISS-085:
// the SQLite path must match the in-memory repo's contract for Burn. The
// in-memory tests have long asserted that after a burn GetOperations still
// returns the mint AND the burn operation (ops[1].IsBurn()) with GetNFT nil,
// but the real repository declared `ON DELETE CASCADE` on nft_operations, so
// TryDeleteNFTIfOwned wiped every operation row for the NFT -- including the
// burn op just persisted -- inside the same transaction, gutting
// `nft history` and GET /api/v1/nft/{id}/operations on the production path.
func TestNFTService_Burn_RetainsAuditTrailOverSQLite(t *testing.T) {
	repo, err := sqlite.NewNFTRepository(filepath.Join(t.TempDir(), "nft_burn.db"))
	if err != nil {
		t.Fatalf("NewNFTRepository: %v", err)
	}
	defer repo.Close()

	svc := nft.NewService(repo, sqlite.NewTxManager(repo.GetDB()))
	chain := &stubChain{}

	ownerPub, ownerPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	minted, err := svc.Mint(nft.NewNFT("Burnable", "desc", "", "", ownerPub, ownerPub), chain)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	ops, err := svc.GetOperations(minted.ID)
	if err != nil {
		t.Fatalf("GetOperations before burn: %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("expected 1 operation after mint, got %d", len(ops))
	}

	if err := svc.Burn(minted.ID, ownerPub, ownerPriv, chain); err != nil {
		t.Fatalf("Burn: %v", err)
	}

	fetched, err := repo.GetNFT(minted.ID)
	if err != nil {
		t.Fatalf("GetNFT after burn: %v", err)
	}
	if fetched != nil {
		t.Fatalf("expected NFT to be gone after burn, got %#v", fetched)
	}

	ops, err = svc.GetOperations(minted.ID)
	if err != nil {
		t.Fatalf("GetOperations after burn: %v", err)
	}
	if len(ops) != 2 {
		t.Fatalf("audit trail must survive burn over real SQLite: expected 2 operations (mint + burn), got %d", len(ops))
	}
	var sawBurn bool
	for _, op := range ops {
		if op.IsBurn() {
			sawBurn = true
		}
	}
	if !sawBurn {
		t.Fatal("burn operation must be retained in the audit trail after burn")
	}
}

// TestNFTRepository_EnsureNoCascadeFKUpgradesLegacyDB verifies existing
// installs are healed: a database whose nft_operations was created with the
// pre-TASK-092 `ON DELETE CASCADE` FK (simulated here with raw SQL, bypassing
// the fixed constructor) must be rebuilt without the FK on the next open,
// preserving its rows, so a delete of the owning NFT no longer wipes the
// operation history.
func TestNFTRepository_EnsureNoCascadeFKUpgradesLegacyDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy_nft.db")

	// Build a legacy DB exactly as the pre-fix createTables did.
	raw, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=ON")
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	for _, q := range []string{
		`PRAGMA foreign_keys=ON`,
		`CREATE TABLE IF NOT EXISTS nfts (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			image_url TEXT,
			token_uri TEXT,
			owner TEXT NOT NULL,
			creator TEXT NOT NULL,
			block_height INTEGER NOT NULL,
			timestamp INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS nft_operations (
			id TEXT PRIMARY KEY,
			nft_id TEXT NOT NULL,
			type TEXT NOT NULL,
			from_addr TEXT,
			to_addr TEXT,
			signature TEXT,
			block_height INTEGER NOT NULL,
			timestamp INTEGER NOT NULL,
			FOREIGN KEY (nft_id) REFERENCES nfts(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_nft_ops_nft_id ON nft_operations(nft_id)`,
		`INSERT OR REPLACE INTO nfts (id, name, description, image_url, token_uri, owner, creator, block_height, timestamp)
		 VALUES ('nft-legacy', 'Legacy', '', '', '', 'owner', 'owner', 1, 1)`,
		`INSERT INTO nft_operations (id, nft_id, type, from_addr, to_addr, signature, block_height, timestamp)
		 VALUES ('op-mint', 'nft-legacy', 'mint', 'owner', 'owner', 'sig', 1, 1)`,
	} {
		if _, err := raw.Exec(q); err != nil {
			raw.Close()
			t.Fatalf("seed legacy schema %q: %v", q, err)
		}
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("close legacy db: %v", err)
	}

	// Reopen through the real constructor: createTables + ensureNoCascadeFK
	// must replace the cascade table and keep the seeded operation.
	repo, err := sqlite.NewNFTRepository(dbPath)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer repo.Close()

	// 1. The FK must be gone.
	rows, err := repo.GetDB().Query(`PRAGMA foreign_key_list(nft_operations)`)
	if err != nil {
		t.Fatalf("inspect FKs: %v", err)
	}
	fkRows := 0
	for rows.Next() {
		fkRows++
	}
	rows.Close()
	if fkRows != 0 {
		t.Fatalf("nft_operations still carries %d foreign key(s) after rebuild", fkRows)
	}

	// 2. Rows survived the rebuild.
	var opCount int
	if err := repo.GetDB().QueryRow(`SELECT COUNT(*) FROM nft_operations WHERE nft_id='nft-legacy' AND id='op-mint'`).Scan(&opCount); err != nil {
		t.Fatalf("count retained op: %v", err)
	}
	if opCount != 1 {
		t.Fatalf("expected seeded operation to survive rebuild, got count=%d", opCount)
	}

	// 3. Deleting the NFT no longer cascades into the operation history.
	if _, err := repo.GetDB().Exec(`DELETE FROM nfts WHERE id='nft-legacy'`); err != nil {
		t.Fatalf("delete nft must not violate FK after rebuild: %v", err)
	}
	var afterDelete int
	if err := repo.GetDB().QueryRow(`SELECT COUNT(*) FROM nft_operations WHERE nft_id='nft-legacy'`).Scan(&afterDelete); err != nil {
		t.Fatalf("count ops after nft delete: %v", err)
	}
	if afterDelete != 1 {
		t.Fatalf("operation history must survive NFT deletion after rebuild, got count=%d", afterDelete)
	}
}

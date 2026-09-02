package sqlite

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pplmx/aurora/internal/domain/nft"
)

// maxNFTHistoryOps bounds the operations GetOperations returns so an unbounded
// reads can't be forced through GET /nft/{id}/history / `nft history`
// (TASK-271, ISS-267).
const maxNFTHistoryOps = 1000

// nftExec abstracts *sql.DB and *sql.Tx so NFTRepository can run either
// against the pool or inside an explicit transaction.
type nftExec interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type NFTRepository struct {
	db     *sql.DB
	dbPath string
	exec   nftExec // nil => use db
}

// q returns the executor this repository reads/writes through: the pooled DB
// by default, or the active *sql.Tx when WithTx was used.
func (r *NFTRepository) q() nftExec {
	if r != nil && r.exec != nil {
		return r.exec
	}
	return r.db
}

// WithTx returns an NFTRepository whose operations run inside the given
// transaction. All reads and writes through the returned repository observe
// the transaction's uncommitted state and participate in its commit/rollback.
func (r *NFTRepository) WithTx(tx *sql.Tx) nft.Repository {
	return &NFTRepository{db: r.db, dbPath: r.dbPath, exec: tx}
}

// GetDB exposes the underlying pool so callers can build a TxManager that
// shares this repository's database file.
func (r *NFTRepository) GetDB() *sql.DB {
	return r.db
}

func NewNFTRepository(dbPath string) (*NFTRepository, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	database, err := sql.Open("sqlite3", DSN(dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	repo := &NFTRepository{
		db:     database,
		dbPath: dbPath,
	}

	if err := repo.createTables(); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return repo, nil
}

func (r *NFTRepository) createTables() error {
	if _, err := r.db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to set WAL mode: %w", err)
	}

	if _, err := r.db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	queries := []string{
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
		// NOTE: nft_operations deliberately carries NO foreign key to nfts.
		// Its rows are an immutable audit trail that must outlive the NFT:
		// Burn deletes the nfts row and the old ON DELETE CASCADE wiped
		// every operation for the NFT inside the same transaction, including
		// the just-saved burn op, so `nft history` / GET /api/v1/nft/{id}/
		// operations came back empty after a successful burn (TASK-092,
		// ISS-085). Existing databases that still have the cascade FK are
		// rebuilt by ensureNoCascadeFK below.
		`CREATE TABLE IF NOT EXISTS nft_operations (
			id TEXT PRIMARY KEY,
			nft_id TEXT NOT NULL,
			type TEXT NOT NULL,
			from_addr TEXT,
			to_addr TEXT,
			signature TEXT,
			block_height INTEGER NOT NULL,
			timestamp INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_nft_owner ON nfts(owner)`,
		`CREATE INDEX IF NOT EXISTS idx_nft_creator ON nfts(creator)`,
		`CREATE INDEX IF NOT EXISTS idx_nft_ops_nft_id ON nft_operations(nft_id)`,
	}

	for _, query := range queries {
		if _, err := r.db.Exec(query); err != nil {
			return err
		}
	}

	return r.ensureNoCascadeFK()
}

// ensureNoCascadeFK upgrades databases created before TASK-092, whose
// nft_operations table was defined with `FOREIGN KEY (nft_id) REFERENCES
// nfts(id) ON DELETE CASCADE`. That cascade destroyed the NFT's whole audit
// trail whenever Burn deleted the nfts row (the burn op it had just persisted
// vanished with the rest). The rebuild is idempotent: once the table has no
// FK in PRAGMA foreign_key_list it is a no-op. SQLite has no ALTER TABLE DROP
// CONSTRAINT, so the table is re-created and copied inside one transaction.
// nft_operations is the CHILD side of the removed FK, so dropping/renaming it
// never triggers a constraint check against nfts (only writes to a child do),
// which means no PRAGMA foreign_keys toggle is needed for the rebuild.
func (r *NFTRepository) ensureNoCascadeFK() error {
	hasFK := false
	rows, err := r.db.Query(`PRAGMA foreign_key_list(nft_operations)`)
	if err != nil {
		return fmt.Errorf("inspect nft_operations foreign keys: %w", err)
	}
	for rows.Next() {
		hasFK = true
		break
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if !hasFK {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin nft_operations rebuild: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	rebuild := []string{
		`CREATE TABLE nft_operations_v2 (
			id TEXT PRIMARY KEY,
			nft_id TEXT NOT NULL,
			type TEXT NOT NULL,
			from_addr TEXT,
			to_addr TEXT,
			signature TEXT,
			block_height INTEGER NOT NULL,
			timestamp INTEGER NOT NULL
		)`,
		`INSERT INTO nft_operations_v2 (id, nft_id, type, from_addr, to_addr, signature, block_height, timestamp)
		 SELECT id, nft_id, type, from_addr, to_addr, signature, block_height, timestamp FROM nft_operations`,
		`DROP TABLE nft_operations`,
		`ALTER TABLE nft_operations_v2 RENAME TO nft_operations`,
		`CREATE INDEX IF NOT EXISTS idx_nft_ops_nft_id ON nft_operations(nft_id)`,
	}
	for _, q := range rebuild {
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("rebuild nft_operations without cascade FK: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit nft_operations rebuild: %w", err)
	}
	return nil
}

func (r *NFTRepository) SaveNFT(n *nft.NFT) error {
	_, err := r.q().Exec(`
		INSERT OR REPLACE INTO nfts (id, name, description, image_url, token_uri, owner, creator, block_height, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		n.ID,
		n.Name,
		n.Description,
		n.ImageURL,
		n.TokenURI,
		base64.StdEncoding.EncodeToString(n.Owner),
		base64.StdEncoding.EncodeToString(n.Creator),
		n.BlockHeight,
		n.Timestamp,
	)
	return err
}

func (r *NFTRepository) GetNFT(id string) (*nft.NFT, error) {
	var name, description, imageURL, tokenURI, ownerB64, creatorB64 string
	var blockHeight, timestamp int64

	err := r.q().QueryRow(`
		SELECT id, name, description, image_url, token_uri, owner, creator, block_height, timestamp
		FROM nfts WHERE id = ?
	`, id).Scan(&id, &name, &description, &imageURL, &tokenURI, &ownerB64, &creatorB64, &blockHeight, &timestamp)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	owner, err := base64.StdEncoding.DecodeString(ownerB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode owner: %w", err)
	}
	creator, err := base64.StdEncoding.DecodeString(creatorB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode creator: %w", err)
	}

	return &nft.NFT{
		ID:          id,
		Name:        name,
		Description: description,
		ImageURL:    imageURL,
		TokenURI:    tokenURI,
		Owner:       owner,
		Creator:     creator,
		BlockHeight: blockHeight,
		Timestamp:   timestamp,
	}, nil
}

func (r *NFTRepository) GetNFTsByOwner(owner []byte, limit, offset int) ([]*nft.NFT, error) {
	ownerB64 := base64.StdEncoding.EncodeToString(owner)
	// SQLite treats a negative LIMIT as unlimited, so 0,0 (the CLI/TUI
	// default) pages the whole collection while the REST layer's bounded
	// limit caps the response (TASK-101, ISS-093).
	if limit <= 0 {
		limit = -1
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.q().Query(`
		SELECT id, name, description, image_url, token_uri, owner, creator, block_height, timestamp
		FROM nfts WHERE owner = ?
		ORDER BY rowid ASC
		LIMIT ? OFFSET ?
	`, ownerB64, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []*nft.NFT
	for rows.Next() {
		var id, name, description, imageURL, tokenURI, ownerStr, creatorB64 string
		var blockHeight, timestamp int64

		if err := rows.Scan(&id, &name, &description, &imageURL, &tokenURI, &ownerStr, &creatorB64, &blockHeight, &timestamp); err != nil {
			return nil, err
		}

		ownerBytes, err := base64.StdEncoding.DecodeString(ownerStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode owner: %w", err)
		}
		creator, err := base64.StdEncoding.DecodeString(creatorB64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode creator: %w", err)
		}

		results = append(results, &nft.NFT{
			ID:          id,
			Name:        name,
			Description: description,
			ImageURL:    imageURL,
			TokenURI:    tokenURI,
			Owner:       ownerBytes,
			Creator:     creator,
			BlockHeight: blockHeight,
			Timestamp:   timestamp,
		})
	}

	return results, rows.Err()
}

func (r *NFTRepository) GetNFTsByCreator(creator []byte) ([]*nft.NFT, error) {
	creatorB64 := base64.StdEncoding.EncodeToString(creator)
	rows, err := r.q().Query(`
		SELECT id, name, description, image_url, token_uri, owner, creator, block_height, timestamp
		FROM nfts WHERE creator = ?
	`, creatorB64)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []*nft.NFT
	for rows.Next() {
		var id, name, description, imageURL, tokenURI, ownerB64, creatorStr string
		var blockHeight, timestamp int64

		if err := rows.Scan(&id, &name, &description, &imageURL, &tokenURI, &ownerB64, &creatorStr, &blockHeight, &timestamp); err != nil {
			return nil, err
		}

		owner, err := base64.StdEncoding.DecodeString(ownerB64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode owner: %w", err)
		}
		creatorBytes, err := base64.StdEncoding.DecodeString(creatorStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode creator: %w", err)
		}

		results = append(results, &nft.NFT{
			ID:          id,
			Name:        name,
			Description: description,
			ImageURL:    imageURL,
			TokenURI:    tokenURI,
			Owner:       owner,
			Creator:     creatorBytes,
			BlockHeight: blockHeight,
			Timestamp:   timestamp,
		})
	}

	return results, rows.Err()
}

func (r *NFTRepository) UpdateNFT(n *nft.NFT) error {
	_, err := r.q().Exec(`
		UPDATE nfts SET name = ?, description = ?, image_url = ?, token_uri = ?, owner = ?, block_height = ?, timestamp = ?
		WHERE id = ?
	`,
		n.Name,
		n.Description,
		n.ImageURL,
		n.TokenURI,
		base64.StdEncoding.EncodeToString(n.Owner),
		n.BlockHeight,
		n.Timestamp,
		n.ID,
	)
	return err
}

// TryTransferOwnership atomically transfers the NFT's owner from
// `from` to `to`. The conditional UPDATE — `WHERE id = ? AND owner
// = ?` — is the atomicity primitive: only the caller that still
// holds the expected owner succeeds. RowsAffected==0 means either
// the NFT does not exist or its owner has moved; in both cases
// the transfer is rejected with nft.ErrOwnershipChanged (or the
// underlying "not found" error if we can disambiguate).
//
// This closes the TOCTOU window in NFTService.Transfer where two
// concurrent transfers both observed the same owner, both
// computed their own recipient, and the last writer silently
// clobbered the other.
func (r *NFTRepository) TryTransferOwnership(nftID string, from, to []byte) error {
	fromB64 := base64.StdEncoding.EncodeToString(from)
	toB64 := base64.StdEncoding.EncodeToString(to)
	res, err := r.q().Exec(`
		UPDATE nfts SET owner = ?
		WHERE id = ? AND owner = ?
	`, toB64, nftID, fromB64)
	if err != nil {
		return fmt.Errorf("try transfer ownership: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("try transfer ownership rows: %w", err)
	}
	if affected == 0 {
		// Disambiguate "missing" from "ownership changed" so the
		// caller can return the right error code.
		existing, err := r.GetNFT(nftID)
		if err != nil {
			return err
		}
		if existing == nil {
			return fmt.Errorf("nft %q not found", nftID)
		}
		return nft.ErrOwnershipChanged
	}
	return nil
}

// TryDeleteNFTIfOwned atomically deletes the NFT if its current
// owner matches `expectedOwner`. Same pattern as
// TryTransferOwnership: a single conditional DELETE statement
// where the row-level lock acquired by SQLite during the DELETE
// guarantees no other writer can sneak in between our ownership
// check and our delete. RowsAffected==0 disambiguates "missing"
// from "ownership moved" via a follow-up read.
func (r *NFTRepository) TryDeleteNFTIfOwned(nftID string, expectedOwner []byte) error {
	ownerB64 := base64.StdEncoding.EncodeToString(expectedOwner)
	res, err := r.q().Exec(`
		DELETE FROM nfts WHERE id = ? AND owner = ?
	`, nftID, ownerB64)
	if err != nil {
		return fmt.Errorf("try delete nft if owned: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("try delete nft if owned rows: %w", err)
	}
	if affected == 0 {
		existing, err := r.GetNFT(nftID)
		if err != nil {
			return err
		}
		if existing == nil {
			return nft.ErrNFTNotFound
		}
		return nft.ErrOwnershipChanged
	}
	return nil
}

func (r *NFTRepository) DeleteNFT(id string) error {
	_, err := r.q().Exec(`DELETE FROM nfts WHERE id = ?`, id)
	return err
}

func (r *NFTRepository) SaveOperation(op *nft.Operation) error {
	var fromB64, toB64, sigB64 string
	if op.From != nil {
		fromB64 = base64.StdEncoding.EncodeToString(op.From)
	}
	if op.To != nil {
		toB64 = base64.StdEncoding.EncodeToString(op.To)
	}
	if op.Signature != nil {
		sigB64 = base64.StdEncoding.EncodeToString(op.Signature)
	}

	_, err := r.q().Exec(`
		INSERT OR REPLACE INTO nft_operations (id, nft_id, type, from_addr, to_addr, signature, block_height, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		op.ID,
		op.NFTID,
		op.Type,
		fromB64,
		toB64,
		sigB64,
		op.BlockHeight,
		op.Timestamp,
	)
	return err
}

func (r *NFTRepository) GetOperations(nftID string) ([]*nft.Operation, error) {
	// Bound the returned operations so a key-holding caller cannot force an
	// unbounded DB scan/response through GET /nft/{id}/history or `nft history`.
	// The rest of the REST surface (token history, NFT list, oracle query) caps
	// ?limit at parse time; this endpoint had no knob, so a fixed cap here
	// closes the same unbounded-read gap (TASK-271, ISS-267). 1000 is beyond
	// any realistic per-NFT transfer/burn trail.
	rows, err := r.q().Query(`
		SELECT id, nft_id, type, from_addr, to_addr, signature, block_height, timestamp
		FROM nft_operations WHERE nft_id = ? ORDER BY timestamp DESC LIMIT ?
	`, nftID, maxNFTHistoryOps)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []*nft.Operation
	for rows.Next() {
		var id, nftID, opType string
		// from_addr/to_addr/signature are nullable columns; scanning NULL into
		// a plain string makes database/sql error and fails the entire history
		// read with a 500 for any legacy/NULL row. NullString keeps them lenient.
		var fromNS, toNS, sigNS sql.NullString
		var blockHeight, timestamp int64

		if err := rows.Scan(&id, &nftID, &opType, &fromNS, &toNS, &sigNS, &blockHeight, &timestamp); err != nil {
			return nil, err
		}

		var from, to, sig []byte
		// Values are self-written base64; a corrupt one leaves that field nil
		// rather than failing the whole history (the columns are nullable by
		// design for pre-history rows).
		if fromNS.Valid && fromNS.String != "" {
			from, _ = base64.StdEncoding.DecodeString(fromNS.String)
		}
		if toNS.Valid && toNS.String != "" {
			to, _ = base64.StdEncoding.DecodeString(toNS.String)
		}
		if sigNS.Valid && sigNS.String != "" {
			sig, _ = base64.StdEncoding.DecodeString(sigNS.String)
		}

		results = append(results, &nft.Operation{
			ID:          id,
			NFTID:       nftID,
			Type:        opType,
			From:        from,
			To:          to,
			Signature:   sig,
			BlockHeight: blockHeight,
			Timestamp:   timestamp,
		})
	}

	return results, rows.Err()
}

func (r *NFTRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

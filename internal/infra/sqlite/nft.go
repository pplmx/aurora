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

type NFTRepository struct {
	db     *sql.DB
	dbPath string
}

func NewNFTRepository(dbPath string) (*NFTRepository, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	database, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=ON", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	repo := &NFTRepository{
		db:     database,
		dbPath: dbPath,
	}

	if err := repo.createTables(); err != nil {
		database.Close()
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
		`CREATE INDEX IF NOT EXISTS idx_nft_owner ON nfts(owner)`,
		`CREATE INDEX IF NOT EXISTS idx_nft_creator ON nfts(creator)`,
		`CREATE INDEX IF NOT EXISTS idx_nft_ops_nft_id ON nft_operations(nft_id)`,
	}

	for _, query := range queries {
		if _, err := r.db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

func (r *NFTRepository) SaveNFT(n *nft.NFT) error {
	_, err := r.db.Exec(`
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

	err := r.db.QueryRow(`
		SELECT id, name, description, image_url, token_uri, owner, creator, block_height, timestamp
		FROM nfts WHERE id = ?
	`, id).Scan(&id, &name, &description, &imageURL, &tokenURI, &ownerB64, &creatorB64, &blockHeight, &timestamp)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	owner, _ := base64.StdEncoding.DecodeString(ownerB64)
	creator, _ := base64.StdEncoding.DecodeString(creatorB64)

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

func (r *NFTRepository) GetNFTsByOwner(owner []byte) ([]*nft.NFT, error) {
	ownerB64 := base64.StdEncoding.EncodeToString(owner)
	rows, err := r.db.Query(`
		SELECT id, name, description, image_url, token_uri, owner, creator, block_height, timestamp
		FROM nfts WHERE owner = ?
	`, ownerB64)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*nft.NFT
	for rows.Next() {
		var id, name, description, imageURL, tokenURI, ownerStr, creatorB64 string
		var blockHeight, timestamp int64

		if err := rows.Scan(&id, &name, &description, &imageURL, &tokenURI, &ownerStr, &creatorB64, &blockHeight, &timestamp); err != nil {
			return nil, err
		}

		ownerBytes, _ := base64.StdEncoding.DecodeString(ownerStr)
		creator, _ := base64.StdEncoding.DecodeString(creatorB64)

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
	rows, err := r.db.Query(`
		SELECT id, name, description, image_url, token_uri, owner, creator, block_height, timestamp
		FROM nfts WHERE creator = ?
	`, creatorB64)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*nft.NFT
	for rows.Next() {
		var id, name, description, imageURL, tokenURI, ownerB64, creatorStr string
		var blockHeight, timestamp int64

		if err := rows.Scan(&id, &name, &description, &imageURL, &tokenURI, &ownerB64, &creatorStr, &blockHeight, &timestamp); err != nil {
			return nil, err
		}

		owner, _ := base64.StdEncoding.DecodeString(ownerB64)
		creatorBytes, _ := base64.StdEncoding.DecodeString(creatorStr)

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
	_, err := r.db.Exec(`
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

func (r *NFTRepository) DeleteNFT(id string) error {
	_, err := r.db.Exec(`DELETE FROM nfts WHERE id = ?`, id)
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

	_, err := r.db.Exec(`
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
	rows, err := r.db.Query(`
		SELECT id, nft_id, type, from_addr, to_addr, signature, block_height, timestamp
		FROM nft_operations WHERE nft_id = ? ORDER BY timestamp DESC
	`, nftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*nft.Operation
	for rows.Next() {
		var id, nftID, opType, fromB64, toB64, sigB64 string
		var blockHeight, timestamp int64

		if err := rows.Scan(&id, &nftID, &opType, &fromB64, &toB64, &sigB64, &blockHeight, &timestamp); err != nil {
			return nil, err
		}

		var from, to, sig []byte
		if fromB64 != "" {
			from, _ = base64.StdEncoding.DecodeString(fromB64)
		}
		if toB64 != "" {
			to, _ = base64.StdEncoding.DecodeString(toB64)
		}
		if sigB64 != "" {
			sig, _ = base64.StdEncoding.DecodeString(sigB64)
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

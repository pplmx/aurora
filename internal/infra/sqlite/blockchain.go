package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pplmx/aurora/internal/domain/blockchain"
)

type BlockchainRepository struct {
	db     *sql.DB
	dbPath string
	chain  *blockchain.BlockChain
}

func NewBlockchainRepository(dbPath string) (*BlockchainRepository, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	database, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	repo := &BlockchainRepository{
		db:     database,
		dbPath: dbPath,
	}

	if err := repo.createTables(); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	chain, err := repo.loadFromDB()
	if err != nil {
		chain = blockchain.NewBlockChain()
	}

	repo.chain = chain
	return repo, nil
}

func (r *BlockchainRepository) createTables() error {
	if _, err := r.db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to set WAL mode: %w", err)
	}

	if _, err := r.db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS blocks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			height INTEGER NOT NULL UNIQUE,
			hash TEXT NOT NULL,
			previous_hash TEXT NOT NULL,
			data TEXT NOT NULL,
			nonce INTEGER NOT NULL,
			created_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_blocks_height ON blocks(height)`,
		`CREATE INDEX IF NOT EXISTS idx_blocks_hash ON blocks(hash)`,
		`CREATE INDEX IF NOT EXISTS idx_blocks_created_at ON blocks(created_at)`,
	}

	for _, query := range queries {
		if _, err := r.db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

func (r *BlockchainRepository) SaveBlock(height int64, block *blockchain.Block) error {
	_, err := r.db.Exec(`
		INSERT OR REPLACE INTO blocks (height, hash, previous_hash, data, nonce, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		height,
		string(block.Hash),
		string(block.PrevHash),
		string(block.Data),
		block.Nonce,
		0,
	)
	return err
}

func (r *BlockchainRepository) GetBlock(height int64) (*blockchain.Block, error) {
	var hash, prevHash, data string
	var nonce int64

	err := r.db.QueryRow(`
		SELECT hash, previous_hash, data, nonce FROM blocks WHERE height = ?
	`, height).Scan(&hash, &prevHash, &data, &nonce)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("block not found at height %d", height)
	}
	if err != nil {
		return nil, err
	}

	return &blockchain.Block{
		Height:    height,
		Hash:      []byte(hash),
		PrevHash:  []byte(prevHash),
		Data:      []byte(data),
		Nonce:     nonce,
		Timestamp: 0,
	}, nil
}

func (r *BlockchainRepository) GetAllBlocks() ([]*blockchain.Block, error) {
	rows, err := r.db.Query("SELECT height, hash, previous_hash, data, nonce FROM blocks ORDER BY height")
	if err != nil {
		return nil, fmt.Errorf("failed to query blocks: %w", err)
	}
	defer rows.Close()

	var blocks []*blockchain.Block
	for rows.Next() {
		var height int64
		var hash, prevHash, data string
		var nonce int64

		if err := rows.Scan(&height, &hash, &prevHash, &data, &nonce); err != nil {
			return nil, fmt.Errorf("failed to scan block: %w", err)
		}

		block := &blockchain.Block{
			Height:    height,
			Hash:      []byte(hash),
			PrevHash:  []byte(prevHash),
			Data:      []byte(data),
			Nonce:     nonce,
			Timestamp: 0,
		}
		blocks = append(blocks, block)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	if len(blocks) == 0 {
		blocks = []*blockchain.Block{blockchain.Genesis()}
	}

	return blocks, nil
}

func (r *BlockchainRepository) GetLotteryRecords() ([]string, error) {
	rows, err := r.db.Query("SELECT data FROM blocks WHERE data != 'Genesis' ORDER BY height")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []string
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		records = append(records, data)
	}

	return records, nil
}

func (r *BlockchainRepository) AddLotteryRecord(data string) (int64, error) {
	if r.chain == nil {
		r.chain = blockchain.NewBlockChain()
	}

	height := r.chain.AddBlock(data)
	block := r.chain.Blocks[len(r.chain.Blocks)-1]

	if err := r.SaveBlock(height, block); err != nil {
		return 0, err
	}

	return height, nil
}

func (r *BlockchainRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

func (r *BlockchainRepository) loadFromDB() (*blockchain.BlockChain, error) {
	blocks, err := r.GetAllBlocks()
	if err != nil {
		return nil, err
	}
	return &blockchain.BlockChain{Blocks: blocks}, nil
}

func (r *BlockchainRepository) Chain() *blockchain.BlockChain {
	return r.chain
}

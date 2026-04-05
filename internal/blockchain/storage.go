package blockchain

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB
var chainInstance *BlockChain

const dbPath = "./data/aurora.db"

// ResetForTest resets the global state for testing
func ResetForTest() {
	if db != nil {
		db.Close()
		db = nil
	}
	chainInstance = nil
	os.RemoveAll("./data")
}

func InitDB() (*sql.DB, error) {
	// Create data directory if not exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open database
	database, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create tables
	if err := createTables(database); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	db = database
	return database, nil
}

func createTables(db *sql.DB) error {
	// Enable WAL mode for better concurrent performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to set WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
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
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

func (chain *BlockChain) SaveBlock(height int, block *Block) error {
	if db == nil {
		var err error
		db, err = InitDB()
		if err != nil {
			return err
		}
	}

	_, err := db.Exec(`
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

func (chain *BlockChain) SaveToDB() error {
	if db == nil {
		var err error
		db, err = InitDB()
		if err != nil {
			return err
		}
	}

	// Clear existing data and re-save all blocks
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Clear blocks table
	if _, err := tx.Exec("DELETE FROM blocks"); err != nil {
		return fmt.Errorf("failed to clear blocks: %w", err)
	}

	// Insert all blocks
	stmt, err := tx.Prepare(`
		INSERT INTO blocks (height, hash, previous_hash, data, nonce, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for i, block := range chain.Blocks {
		if _, err := stmt.Exec(
			i,
			string(block.Hash),
			string(block.PrevHash),
			string(block.Data),
			block.Nonce,
			0, // created_at timestamp
		); err != nil {
			return fmt.Errorf("failed to insert block %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func LoadFromDB() (*BlockChain, error) {
	database, err := InitDB()
	if err != nil {
		return nil, err
	}

	rows, err := database.Query("SELECT height, hash, previous_hash, data, nonce FROM blocks ORDER BY height")
	if err != nil {
		return nil, fmt.Errorf("failed to query blocks: %w", err)
	}
	defer rows.Close()

	chain := &BlockChain{Blocks: []*Block{}}

	for rows.Next() {
		var height int
		var hash, prevHash, data string
		var nonce int

		if err := rows.Scan(&height, &hash, &prevHash, &data, &nonce); err != nil {
			return nil, fmt.Errorf("failed to scan block: %w", err)
		}

		block := &Block{
			Hash:     []byte(hash),
			PrevHash: []byte(prevHash),
			Data:     []byte(data),
			Nonce:    nonce,
		}
		chain.Blocks = append(chain.Blocks, block)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// If no blocks, create genesis
	if len(chain.Blocks) == 0 {
		chain.Blocks = append(chain.Blocks, Genesis())
	}

	return chain, nil
}

func CloseDB() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

func DBPath() string {
	return dbPath
}

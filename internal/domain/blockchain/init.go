package blockchain

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pplmx/aurora/internal/logger"
)

var (
	instance   *BlockChain
	dbInstance *sql.DB
	once       sync.Once
)

const defaultDBPath = "./data/aurora.db"

func DBPath() string {
	dir := filepath.Dir(defaultDBPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ""
	}
	return defaultDBPath
}

func InitDB() (*sql.DB, error) {
	if dbInstance != nil {
		return dbInstance, nil
	}

	dir := filepath.Dir(defaultDBPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", defaultDBPath+"?_foreign_keys=ON")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	dbInstance = db
	return dbInstance, nil
}

func InitBlockChain() *BlockChain {
	once.Do(func() {
		chain := &BlockChain{[]*Block{Genesis()}}

		if db, err := InitDB(); err == nil {
			if _, err := db.Exec(`
				CREATE TABLE IF NOT EXISTS blocks (
					height INTEGER PRIMARY KEY,
					hash TEXT NOT NULL,
					previous_hash TEXT NOT NULL,
					data TEXT NOT NULL,
					nonce INTEGER NOT NULL,
					timestamp INTEGER NOT NULL,
					created_at INTEGER NOT NULL
				)
			`); err != nil {
				logger.Error().Err(err).Msg("Failed to create blocks table")
			}

			rows, err := db.Query("SELECT height, hash, previous_hash, data, nonce FROM blocks ORDER BY height")
			if err == nil {
				for rows.Next() {
					var block Block
					var hash, prevHash, data string
					if err := rows.Scan(&block.Height, &hash, &prevHash, &data, &block.Nonce); err == nil {
						block.Hash = []byte(hash)
						block.PrevHash = []byte(prevHash)
						block.Data = []byte(data)
						chain.Blocks = append(chain.Blocks, &block)
					}
				}
				rows.Close()
			}

			if len(chain.Blocks) <= 1 {
				stmt, err := db.Prepare("INSERT INTO blocks (height, hash, previous_hash, data, nonce, timestamp, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)")
				if err == nil {
					block := chain.Blocks[0]
					_, _ = stmt.Exec(block.Height, string(block.Hash), string(block.PrevHash), string(block.Data), block.Nonce, block.Timestamp, block.Timestamp)
				}
			}
		}

		instance = chain
	})

	return instance
}

func GetBlockChain() *BlockChain {
	if instance == nil {
		return InitBlockChain()
	}
	return instance
}

func ResetForTest() {
	if dbInstance != nil {
		dbInstance.Close()
		dbInstance = nil
	}
	instance = nil
	once = sync.Once{}
	os.RemoveAll("./data")
}

func (c *BlockChain) AddLotteryRecord(data string) (int64, error) {
	height := c.AddBlock(data)
	return height, nil
}

func Close() error {
	if dbInstance != nil {
		return dbInstance.Close()
	}
	return nil
}

package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/utils"
	_ "modernc.org/sqlite"
)

func arrayToJSON(arr []string) string {
	data, err := json.Marshal(arr)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func jsonToArray(data string) []string {
	var arr []string
	if err := json.Unmarshal([]byte(data), &arr); err != nil {
		return nil
	}
	return arr
}

type lotteryExec interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type LotteryRepository struct {
	db     *sql.DB
	dbPath string
	exec   lotteryExec
}

// lotteryNewestFirst is the one ordering every newest-first lottery read uses.
//
// Timestamps have one-second resolution, so two draws created in the same
// second tie. SQLite then returns them in index/rowid (insertion) order, which
// put the older draw first and made a page window able to repeat or skip a row
// between two requests. block_height carries the chain append order and breaks
// the tie newest-first; the id makes the order total when imported records
// share both a timestamp and a height.
const lotteryNewestFirst = `ORDER BY l.timestamp DESC, l.block_height DESC, l.id`

func NewLotteryRepository(dbPath string) (*LotteryRepository, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	database, err := sql.Open("sqlite", DSN(dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	repo := &LotteryRepository{
		db:     database,
		dbPath: dbPath,
	}

	if err := repo.createTables(); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}
	if err := utils.RestrictSQLiteFiles(dbPath); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("failed to secure database files: %w", err)
	}

	return repo, nil
}

// DBPath reports the SQLite file this repository writes. Export needs it to
// refuse a destination that aliases the live database: replacing that file
// with a JSON export destroys the store, even though the open handle keeps
// serving reads until the process exits.
func (r *LotteryRepository) DBPath() string { return r.dbPath }

// GetDB exposes the underlying pool for callers that need to share a
// blockchain transaction with the repository.
func (r *LotteryRepository) GetDB() *sql.DB {
	return r.db
}

func (r *LotteryRepository) q() lotteryExec {
	if r.exec != nil {
		return r.exec
	}
	return r.db
}

// WithTx returns a repository whose operations participate in tx.
func (r *LotteryRepository) WithTx(tx *sql.Tx) lottery.Repository {
	return &LotteryRepository{db: r.db, dbPath: r.dbPath, exec: tx}
}

// WithTransaction runs fn in one write transaction on the repository's DB.
// The lottery use case passes this repository to the blockchain transaction
// writer so the record row and its audit block share one commit/rollback.
func (r *LotteryRepository) WithTransaction(fn func(tx *sql.Tx) error) error {
	return NewTxManager(r.db).WithTransaction(fn)
}

func (r *LotteryRepository) createTables() error {
	if _, err := r.db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to set WAL mode: %w", err)
	}

	if _, err := r.db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS lottery_records (
			id TEXT PRIMARY KEY,
			block_height INTEGER NOT NULL,
			seed TEXT NOT NULL,
			participants TEXT NOT NULL,
			winners TEXT NOT NULL,
			winner_addresses TEXT NOT NULL,
			vrf_proof TEXT NOT NULL,
			vrf_output TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			verified INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS lottery_vrf_keys (
			record_id TEXT PRIMARY KEY,
			public_key TEXT NOT NULL,
			FOREIGN KEY (record_id) REFERENCES lottery_records(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_lottery_block_height ON lottery_records(block_height)`,
		// The composite recency index is the ordering GetAll uses and its
		// leading timestamp column serves every timestamp-only lookup, so the
		// single-column index is dropped rather than maintained twice
		// (migration 000007 does the same).
		`DROP INDEX IF EXISTS idx_lottery_timestamp`,
		`CREATE INDEX IF NOT EXISTS idx_lottery_recency ON lottery_records(timestamp DESC, block_height DESC, id)`,
	}

	for _, query := range queries {
		if _, err := r.db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

func (r *LotteryRepository) Save(record *lottery.LotteryRecord) error {
	if r.exec != nil {
		return saveLotteryRecord(r.exec, record)
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	if err := saveLotteryRecord(tx, record); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func saveLotteryRecord(exec lotteryExec, record *lottery.LotteryRecord) error {
	participantsJSON := arrayToJSON(record.Participants)
	winnersJSON := arrayToJSON(record.Winners)
	winnerAddressesJSON := arrayToJSON(record.WinnerAddresses)

	verified := 0
	if record.Verified {
		verified = 1
	}

	_, err := exec.Exec(`
		INSERT OR REPLACE INTO lottery_records
		(id, block_height, seed, participants, winners, winner_addresses, vrf_proof, vrf_output, timestamp, verified)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		record.ID,
		record.BlockHeight,
		record.Seed,
		participantsJSON,
		winnersJSON,
		winnerAddressesJSON,
		record.VRFProof,
		record.VRFOutput,
		record.Timestamp,
		verified,
	)
	if err != nil {
		return err
	}

	if record.VRFPublicKey == "" {
		_, err = exec.Exec(`DELETE FROM lottery_vrf_keys WHERE record_id = ?`, record.ID)
		return err
	}
	_, err = exec.Exec(`
		INSERT INTO lottery_vrf_keys (record_id, public_key)
		VALUES (?, ?)
		ON CONFLICT(record_id) DO UPDATE SET public_key = excluded.public_key
	`, record.ID, record.VRFPublicKey)
	return err
}

func (r *LotteryRepository) GetByID(id string) (*lottery.LotteryRecord, error) {
	var seed, participantsJSON, winnersJSON, winnerAddressesJSON, vrfProof, vrfOutput, vrfPublicKey string
	var blockHeight, timestamp int64
	var verified int

	err := r.q().QueryRow(`
		SELECT l.id, l.block_height, l.seed, l.participants, l.winners, l.winner_addresses,
		       l.vrf_proof, l.vrf_output, COALESCE(k.public_key, ''), l.timestamp, l.verified
		FROM lottery_records l
		LEFT JOIN lottery_vrf_keys k ON k.record_id = l.id
		WHERE l.id = ?
	`, id).Scan(&id, &blockHeight, &seed, &participantsJSON, &winnersJSON, &winnerAddressesJSON, &vrfProof, &vrfOutput, &vrfPublicKey, &timestamp, &verified)

	if err == sql.ErrNoRows {
		return nil, lottery.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	participants := jsonToArray(participantsJSON)
	winners := jsonToArray(winnersJSON)
	winnerAddresses := jsonToArray(winnerAddressesJSON)

	return &lottery.LotteryRecord{
		ID:              id,
		BlockHeight:     blockHeight,
		Seed:            seed,
		Participants:    participants,
		Winners:         winners,
		WinnerAddresses: winnerAddresses,
		VRFProof:        vrfProof,
		VRFOutput:       vrfOutput,
		VRFPublicKey:    vrfPublicKey,
		Timestamp:       timestamp,
		Verified:        verified == 1,
	}, nil
}

func (r *LotteryRepository) GetAll() ([]*lottery.LotteryRecord, error) {
	return r.List(0, 0)
}

func (r *LotteryRepository) Count() (int, error) {
	var count int
	err := r.q().QueryRow(`SELECT COUNT(*) FROM lottery_records`).Scan(&count)
	return count, err
}

func (r *LotteryRepository) List(limit, offset int) ([]*lottery.LotteryRecord, error) {
	if limit <= 0 {
		limit = -1
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.q().Query(`
		SELECT l.id, l.block_height, l.seed, l.participants, l.winners, l.winner_addresses,
		       l.vrf_proof, l.vrf_output, COALESCE(k.public_key, ''), l.timestamp, l.verified
		FROM lottery_records l
		LEFT JOIN lottery_vrf_keys k ON k.record_id = l.id
		`+lotteryNewestFirst+`
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var records []*lottery.LotteryRecord
	for rows.Next() {
		var id, seed, participantsJSON, winnersJSON, winnerAddressesJSON, vrfProof, vrfOutput, vrfPublicKey string
		var blockHeight, timestamp int64
		var verified int

		if err := rows.Scan(&id, &blockHeight, &seed, &participantsJSON, &winnersJSON, &winnerAddressesJSON, &vrfProof, &vrfOutput, &vrfPublicKey, &timestamp, &verified); err != nil {
			return nil, err
		}

		participants := jsonToArray(participantsJSON)
		winners := jsonToArray(winnersJSON)
		winnerAddresses := jsonToArray(winnerAddressesJSON)

		records = append(records, &lottery.LotteryRecord{
			ID:              id,
			BlockHeight:     blockHeight,
			Seed:            seed,
			Participants:    participants,
			Winners:         winners,
			WinnerAddresses: winnerAddresses,
			VRFProof:        vrfProof,
			VRFOutput:       vrfOutput,
			VRFPublicKey:    vrfPublicKey,
			Timestamp:       timestamp,
			Verified:        verified == 1,
		})
	}

	return records, rows.Err()
}

func (r *LotteryRepository) GetByBlockHeight(height int64) ([]*lottery.LotteryRecord, error) {
	rows, err := r.q().Query(`
		SELECT l.id, l.block_height, l.seed, l.participants, l.winners, l.winner_addresses,
		       l.vrf_proof, l.vrf_output, COALESCE(k.public_key, ''), l.timestamp, l.verified
		FROM lottery_records l
		LEFT JOIN lottery_vrf_keys k ON k.record_id = l.id
		WHERE l.block_height = ?
		`+lotteryNewestFirst+`
	`, height)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var records []*lottery.LotteryRecord
	for rows.Next() {
		var id, seed, participantsJSON, winnersJSON, winnerAddressesJSON, vrfProof, vrfOutput, vrfPublicKey string
		var blockHeight, timestamp int64
		var verified int

		if err := rows.Scan(&id, &blockHeight, &seed, &participantsJSON, &winnersJSON, &winnerAddressesJSON, &vrfProof, &vrfOutput, &vrfPublicKey, &timestamp, &verified); err != nil {
			return nil, err
		}

		participants := jsonToArray(participantsJSON)
		winners := jsonToArray(winnersJSON)
		winnerAddresses := jsonToArray(winnerAddressesJSON)

		records = append(records, &lottery.LotteryRecord{
			ID:              id,
			BlockHeight:     blockHeight,
			Seed:            seed,
			Participants:    participants,
			Winners:         winners,
			WinnerAddresses: winnerAddresses,
			VRFProof:        vrfProof,
			VRFOutput:       vrfOutput,
			VRFPublicKey:    vrfPublicKey,
			Timestamp:       timestamp,
			Verified:        verified == 1,
		})
	}

	return records, rows.Err()
}

func (r *LotteryRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

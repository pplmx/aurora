package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pplmx/aurora/internal/domain/lottery"
)

func arrayToJSON(arr []string) string {
	data, _ := json.Marshal(arr)
	return string(data)
}

func jsonToArray(data string) []string {
	var arr []string
	_ = json.Unmarshal([]byte(data), &arr)
	return arr
}

type LotteryRepository struct {
	db     *sql.DB
	dbPath string
}

func NewLotteryRepository(dbPath string) (*LotteryRepository, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	database, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=ON", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	repo := &LotteryRepository{
		db:     database,
		dbPath: dbPath,
	}

	if err := repo.createTables(); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return repo, nil
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
		`CREATE INDEX IF NOT EXISTS idx_lottery_block_height ON lottery_records(block_height)`,
		`CREATE INDEX IF NOT EXISTS idx_lottery_timestamp ON lottery_records(timestamp)`,
	}

	for _, query := range queries {
		if _, err := r.db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

func (r *LotteryRepository) Save(record *lottery.LotteryRecord) error {
	participantsJSON := arrayToJSON(record.Participants)
	winnersJSON := arrayToJSON(record.Winners)
	winnerAddressesJSON := arrayToJSON(record.WinnerAddresses)

	verified := 0
	if record.Verified {
		verified = 1
	}

	_, err := r.db.Exec(`
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
	return err
}

func (r *LotteryRepository) GetByID(id string) (*lottery.LotteryRecord, error) {
	var seed, participantsJSON, winnersJSON, winnerAddressesJSON, vrfProof, vrfOutput string
	var blockHeight, timestamp int64
	var verified int

	err := r.db.QueryRow(`
		SELECT id, block_height, seed, participants, winners, winner_addresses, vrf_proof, vrf_output, timestamp, verified
		FROM lottery_records WHERE id = ?
	`, id).Scan(&id, &blockHeight, &seed, &participantsJSON, &winnersJSON, &winnerAddressesJSON, &vrfProof, &vrfOutput, &timestamp, &verified)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("lottery record not found: %s", id)
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
		Timestamp:       timestamp,
		Verified:        verified == 1,
	}, nil
}

func (r *LotteryRepository) GetAll() ([]*lottery.LotteryRecord, error) {
	rows, err := r.db.Query(`
		SELECT id, block_height, seed, participants, winners, winner_addresses, vrf_proof, vrf_output, timestamp, verified
		FROM lottery_records ORDER BY timestamp DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*lottery.LotteryRecord
	for rows.Next() {
		var id, seed, participantsJSON, winnersJSON, winnerAddressesJSON, vrfProof, vrfOutput string
		var blockHeight, timestamp int64
		var verified int

		if err := rows.Scan(&id, &blockHeight, &seed, &participantsJSON, &winnersJSON, &winnerAddressesJSON, &vrfProof, &vrfOutput, &timestamp, &verified); err != nil {
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
			Timestamp:       timestamp,
			Verified:        verified == 1,
		})
	}

	return records, rows.Err()
}

func (r *LotteryRepository) GetByBlockHeight(height int64) ([]*lottery.LotteryRecord, error) {
	rows, err := r.db.Query(`
		SELECT id, block_height, seed, participants, winners, winner_addresses, vrf_proof, vrf_output, timestamp, verified
		FROM lottery_records WHERE block_height = ?
	`, height)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*lottery.LotteryRecord
	for rows.Next() {
		var id, seed, participantsJSON, winnersJSON, winnerAddressesJSON, vrfProof, vrfOutput string
		var blockHeight, timestamp int64
		var verified int

		if err := rows.Scan(&id, &blockHeight, &seed, &participantsJSON, &winnersJSON, &winnerAddressesJSON, &vrfProof, &vrfOutput, &timestamp, &verified); err != nil {
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

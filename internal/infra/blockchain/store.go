package blockchain

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

type blockInserter interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func insertBlock(exec blockInserter, block *Block) error {
	_, err := exec.Exec(`
		INSERT INTO blocks (height, hash, previous_hash, data, nonce, timestamp, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, block.Height, string(block.Hash), string(block.PrevHash), string(block.Data), block.Nonce, block.Timestamp, block.Timestamp)
	if err != nil {
		if sqliteIsUniqueViolation(err) {
			return ErrHeightConflict
		}
		return err
	}
	return nil
}

// removeBlock deletes only the exact persisted block supplied by the nil-tx
// rollback path. Matching both height and hash prevents cleanup from deleting a
// different block that another process committed at the same height.
func removeBlock(db *sql.DB, block *Block) error {
	result, err := db.Exec(`DELETE FROM blocks WHERE height = ? AND hash = ?`, block.Height, string(block.Hash))
	if err != nil {
		return err
	}
	removed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if removed != 1 {
		return fmt.Errorf("blockchain: expected to remove one persisted block at height %d, removed %d", block.Height, removed)
	}
	return nil
}

// AddRecord appends a lottery or oracle audit record to the chain and
// returns its height. Unlike AddBlock, it stamps the true height into the
// record's JSON `block_height` field immediately before the block is mined,
// so the immutable on-chain payload is self-describing.
//
// The app-layer callers (lottery create, oracle fetch, the scheduler, the
// desktop) serialized their record while its height was still 0, called
// AddRecord, and only then set BlockHeight from the return value — so
// the permanent chain copy claimed "block_height":0 and a consumer replaying
// the chain could never correlate a block to its record (ISS-097). Stamping
// here, under the same lock that assigns the height, fixes every caller at
// the one point they all funnel through.
func (c *Chain) AddRecord(data string) (int64, error) {
	return c.appendBlock(data, func(height int64) string {
		stamped, err := stampRecordHeight(data, height)
		if err != nil {
			return "" // not JSON (or unreadable) — store unchanged
		}
		return stamped
	})
}

// stampRecordHeight rewrites the block_height field of a JSON audit record to
// the given height. Non-JSON payloads (e.g. tests passing bare strings) error
// and are stored unchanged. RawMessage keeps every unrelated value exact while
// replacing only block_height; decoding through map[string]any would round
// integers above 2^53. Re-marshalling drops the original key order, which is
// immaterial for an opaque chain payload.
func stampRecordHeight(data string, height int64) (string, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(data), &fields); err != nil {
		return "", err
	}
	if fields == nil {
		return "", errors.New("record must be a JSON object")
	}
	fields["block_height"] = json.RawMessage(strconv.FormatInt(height, 10))
	b, err := json.Marshal(fields)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

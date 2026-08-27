package blockchain

import (
	"encoding/json"
	"testing"
)

// TestAddLotteryRecord_StampsBlockHeight is the regression test for the
// on-chain audit defect (ISS-097): app-layer callers serialized their record
// with block_height=0 (the height is only known after the append) and stamped
// only their local DB copy, leaving the immutable chain payload claiming
// height 0 forever. AddLotteryRecord must stamp the true height into the JSON
// before the block is mined.
func TestAddLotteryRecord_StampsBlockHeight(t *testing.T) {
	c := NewBlockChain()

	payload := `{"id":"abc","block_height":0,"seed":"s","winners":["Alice"]}`
	height, err := c.AddLotteryRecord(payload)
	if err != nil {
		t.Fatalf("AddLotteryRecord() error = %v", err)
	}
	if height != 1 {
		t.Fatalf("AddLotteryRecord() height = %d, want 1", height)
	}

	var m map[string]any
	if err := json.Unmarshal(c.Blocks[1].Data, &m); err != nil {
		t.Fatalf("chain block payload is not valid JSON: %v", err)
	}
	if got, ok := m["block_height"].(float64); !ok || int64(got) != height {
		t.Fatalf("chain block block_height = %v, want %d", m["block_height"], height)
	}

	// A second record on the same chain gets its own true height.
	height2, err := c.AddLotteryRecord(`{"id":"def","block_height":0}`)
	if err != nil {
		t.Fatalf("AddLotteryRecord() #2 error = %v", err)
	}
	if height2 != 2 {
		t.Fatalf("AddLotteryRecord() #2 height = %d, want 2", height2)
	}
	var m2 map[string]any
	if err := json.Unmarshal(c.Blocks[2].Data, &m2); err != nil {
		t.Fatalf("second block payload is not valid JSON: %v", err)
	}
	if got, ok := m2["block_height"].(float64); !ok || int64(got) != 2 {
		t.Fatalf("second block block_height = %v, want 2", m2["block_height"])
	}
}

// TestAddLotteryRecord_NonJSONStoredUnchanged ensures plain-text payloads
// (tests, legacy callers) still append exactly as given — stamping only
// applies to JSON records that carry a block_height field.
func TestAddLotteryRecord_NonJSONStoredUnchanged(t *testing.T) {
	c := NewBlockChain()
	height, err := c.AddLotteryRecord("L-1")
	if err != nil {
		t.Fatalf("AddLotteryRecord() error = %v", err)
	}
	if height != 1 {
		t.Fatalf("AddLotteryRecord() height = %d, want 1", height)
	}
	if string(c.Blocks[1].Data) != "L-1" {
		t.Fatalf("non-JSON payload was rewritten: %q", c.Blocks[1].Data)
	}
}

// TestStampRecordHeight covers the JSON rewrite helper directly, including
// preservation of every other field.
func TestStampRecordHeight(t *testing.T) {
	in := `{"id":"abc","block_height":0,"seed":"s","winners":["Alice","Bob"],"timestamp":1700000000}`
	out, err := stampRecordHeight(in, 7)
	if err != nil {
		t.Fatalf("stampRecordHeight() error = %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("stamped payload is not valid JSON: %v", err)
	}
	if got, ok := m["block_height"].(float64); !ok || int64(got) != 7 {
		t.Fatalf("block_height = %v, want 7", m["block_height"])
	}
	if m["id"] != "abc" || m["seed"] != "s" {
		t.Fatalf("unrelated fields not preserved: %v", m)
	}
	if w, ok := m["winners"].([]any); !ok || len(w) != 2 || w[0] != "Alice" {
		t.Fatalf("winners array not preserved: %v", m["winners"])
	}

	if _, err := stampRecordHeight("not-json", 1); err == nil {
		t.Fatal("stampRecordHeight() should error on non-JSON input")
	}
}

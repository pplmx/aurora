// Verifying the chain: the integrity report and the proof-of-work re-check
// behind it. This is the one place a stored block is re-derived rather than
// trusted, so it lives apart from the append path that produced it.
package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"

	"github.com/pplmx/aurora/internal/ledger"
)

// IntegrityReport is the pure ledger type; the alias keeps VerifyIntegrity's
// public return type stable.
type IntegrityReport = ledger.IntegrityReport

// VerifyIntegrity cryptographically verifies the whole chain: heights must be
// contiguous from genesis, every block must carry a non-empty hash, each
// block's PrevHash must link to the previous block's hash, and each block's
// stored hash must be a valid proof-of-work over its fields. It returns a
// report with the first broken index (-1 if valid).
//
// This is the tamper-evidence surface behind the "blockchain-based" guarantees
// of the module (v1.25): operators can prove the ledger is internally consistent.
func (c *Chain) VerifyIntegrity() *IntegrityReport {
	if c == nil {
		return &IntegrityReport{Valid: false, FirstBrokenIndex: 0, FirstBrokenReason: "nil chain"}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	blocks := c.Blocks
	n := len(blocks)
	// The head identifies the state even when a block in the middle broke: a
	// support comparison starts from the last hash either way.
	head := ""
	if n > 0 && blocks[n-1] != nil {
		head = hex.EncodeToString(blocks[n-1].Hash)
	}
	for i, b := range blocks {
		if b == nil {
			return &IntegrityReport{Valid: false, Length: n, Head: head, FirstBrokenIndex: i, FirstBrokenReason: "nil block"}
		}
		if b.Height != int64(i) {
			return &IntegrityReport{Valid: false, Length: n, Head: head, FirstBrokenIndex: i, FirstBrokenReason: "height discontinuity"}
		}
		if len(b.Hash) == 0 {
			return &IntegrityReport{Valid: false, Length: n, Head: head, FirstBrokenIndex: i, FirstBrokenReason: "empty hash"}
		}
		if i > 0 && !bytes.Equal(b.PrevHash, blocks[i-1].Hash) {
			return &IntegrityReport{Valid: false, Length: n, Head: head, FirstBrokenIndex: i, FirstBrokenReason: "prev-hash chain break"}
		}
		if ok, reason := verifyBlockPoW(b); !ok {
			return &IntegrityReport{Valid: false, Length: n, Head: head, FirstBrokenIndex: i, FirstBrokenReason: reason}
		}
	}
	return &IntegrityReport{Valid: true, Length: n, Head: head, FirstBrokenIndex: -1}
}

// verifyBlockPoW confirms the block's stored hash is a valid proof-of-work over
// its fields (PrevHash + Data + nonce + difficulty): the recomputed digest must
// meet the target AND equal the stored hash.
func verifyBlockPoW(b *Block) (bool, string) {
	pow := NewProofOfWork(b)
	if !pow.Validate() {
		return false, "proof of work target not met"
	}
	digest := sha256.Sum256(pow.InitNonce(int(b.Nonce)))
	if !bytes.Equal(digest[:], b.Hash) {
		return false, "stored hash does not match recomputed proof"
	}
	return true, ""
}

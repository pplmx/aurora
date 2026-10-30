package ledger

// BlockEntry is one recorded payload with the height and timestamp of the
// block holding it.
//
// Timestamp is the block's own creation time, not the payload's: a payload may
// carry its own timestamp (lottery and oracle records do), but token and NFT
// records do not, so the block is the only clock every entry shares. Readers
// that show "when" use this rather than parsing a payload shape.
type BlockEntry struct {
	Height    int64
	Timestamp int64
	Data      string
}

// IntegrityReport summarises an integrity verification of the chain.
type IntegrityReport struct {
	Valid  bool `json:"valid"`
	Length int  `json:"length"`
	// Head is the newest block's hash in hex: the chain's identity beside its
	// size, so two installations can confirm they follow the same chain.
	Head              string `json:"head,omitempty"`
	FirstBrokenIndex  int    `json:"first_broken_index,omitempty"`
	FirstBrokenReason string `json:"first_broken_reason,omitempty"`
}

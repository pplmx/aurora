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
	Head string `json:"head,omitempty"`
	// FirstBrokenIndex is -1 when the chain is intact and the index of the first
	// break otherwise. It carries no omitempty: index 0 is a real answer (the
	// genesis block is the one that broke), and omitting it made a broken chain
	// report {"valid":false} with no index at all — the one field a caller needs
	// to say where to look.
	FirstBrokenIndex  int    `json:"first_broken_index"`
	FirstBrokenReason string `json:"first_broken_reason,omitempty"`
}

package ledger

// BlockEntry is one recorded payload with the height of the block holding it.
type BlockEntry struct {
	Height int64
	Data   string
}

// IntegrityReport summarises an integrity verification of the chain.
type IntegrityReport struct {
	Valid             bool   `json:"valid"`
	Length            int    `json:"length"`
	FirstBrokenIndex  int    `json:"first_broken_index,omitempty"`
	FirstBrokenReason string `json:"first_broken_reason,omitempty"`
}

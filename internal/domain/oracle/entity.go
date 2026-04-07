// Package oracle provides data oracle functionality for fetching external
// data (e.g., BTC/ETH prices) and recording it on-chain.
package oracle

type OracleData struct {
	ID          string `json:"id"`
	SourceID    string `json:"source_id"`
	Value       string `json:"value"`
	RawResponse string `json:"raw_response"`
	Timestamp   int64  `json:"timestamp"`
	BlockHeight int64  `json:"block_height"`
}

type DataSource struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	Type      string `json:"type"`
	Method    string `json:"method"`
	Headers   string `json:"headers"`
	Path      string `json:"path"`
	Interval  int    `json:"interval"`
	Enabled   bool   `json:"enabled"`
	CreatedAt int64  `json:"created_at"`
}

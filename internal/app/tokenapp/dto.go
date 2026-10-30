package tokenapp

type CreateTokenRequest struct {
	Name        string
	Symbol      string
	TotalSupply string
	Owner       string
	// Decimals is the optional decimal places. nil means the field was omitted
	// (use the domain default of 8); a non-nil value is explicit, including 0.
	Decimals *int8
}

type CreateTokenResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	TotalSupply string `json:"total_supply"`
	Decimals    int8   `json:"decimals"`
	Owner       string `json:"owner"`
	// The flags ride along so the REST info/list and the CLI's `token info`
	// answer with one shape; a create always produces a mintable, burnable
	// token, but a later revision of the domain may not.
	Mintable bool `json:"mintable"`
	Burnable bool `json:"burnable"`
}

type MintRequest struct {
	TokenID    string
	To         string
	Amount     string
	PrivateKey string
}

type MintResponse struct {
	ID          string `json:"id"`
	TokenID     string `json:"token_id"`
	To          string `json:"to"`
	Amount      string `json:"amount"`
	Timestamp   int64  `json:"timestamp"`
	BlockHeight int64  `json:"block_height"`
}

type TransferRequest struct {
	TokenID    string
	From       string
	To         string
	Amount     string
	PrivateKey string
}

type TransferResponse struct {
	ID      string `json:"id"`
	TokenID string `json:"token_id"`
	From    string `json:"from"`
	To      string `json:"to"`
	Amount  string `json:"amount"`
	// Nonce is the replay-protection sequence number the transfer was signed
	// with. The desktop history table shows it, and without it every row read
	// as nonce 0.
	Nonce       uint64 `json:"nonce"`
	Timestamp   int64  `json:"timestamp"`
	BlockHeight int64  `json:"block_height"`
}

type TransferFromRequest struct {
	TokenID    string
	Owner      string
	To         string
	Amount     string
	Spender    string
	SpenderKey string
}

type TransferFromResponse struct {
	ID          string `json:"id"`
	TokenID     string `json:"token_id"`
	From        string `json:"from"`
	To          string `json:"to"`
	Amount      string `json:"amount"`
	Timestamp   int64  `json:"timestamp"`
	BlockHeight int64  `json:"block_height"`
}

type ApproveRequest struct {
	TokenID    string
	Owner      string
	Spender    string
	Amount     string
	PrivateKey string
}

type ApproveResponse struct {
	ID        string `json:"id"`
	TokenID   string `json:"token_id"`
	Owner     string `json:"owner"`
	Spender   string `json:"spender"`
	Amount    string `json:"amount"`
	Timestamp int64  `json:"timestamp"`
}

type BurnRequest struct {
	TokenID    string
	From       string
	Amount     string
	PrivateKey string
}

type BurnResponse struct {
	ID          string `json:"id"`
	TokenID     string `json:"token_id"`
	From        string `json:"from"`
	Amount      string `json:"amount"`
	Timestamp   int64  `json:"timestamp"`
	BlockHeight int64  `json:"block_height"`
}

type BalanceRequest struct {
	TokenID string
	Owner   string
}

type BalanceResponse struct {
	TokenID string `json:"token_id"`
	Owner   string `json:"owner"`
	Amount  string `json:"amount"`
}

type AllowanceRequest struct {
	TokenID string
	Owner   string
	Spender string
}

type AllowanceResponse struct {
	TokenID string `json:"token_id"`
	Owner   string `json:"owner"`
	Spender string `json:"spender"`
	Amount  string `json:"amount"`
}

type HistoryRequest struct {
	TokenID string
	// Owner is optional. Empty selects the token-wide transfer stream.
	Owner  string
	Limit  int
	Offset int
}

package token

type CreateTokenRequest struct {
	Name        string
	Symbol      string
	TotalSupply string
	Owner       string
}

type CreateTokenResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	TotalSupply string `json:"total_supply"`
	Decimals    int8   `json:"decimals"`
	Owner       string `json:"owner"`
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
	ID          string `json:"id"`
	TokenID     string `json:"token_id"`
	From        string `json:"from"`
	To          string `json:"to"`
	Amount      string `json:"amount"`
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
	Owner   string
	Limit   int
	Offset  int
}

type HistoryResponse struct {
	Transfers []TransferResponse `json:"transfers"`
}

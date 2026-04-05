package token

type CreateTokenRequest struct {
	Name        string
	Symbol      string
	TotalSupply string
	Owner       string
}

type CreateTokenResponse struct {
	ID          string
	Name        string
	Symbol      string
	TotalSupply string
	Decimals    int8
	Owner       string
}

type MintRequest struct {
	TokenID    string
	To         string
	Amount     string
	PrivateKey string
}

type MintResponse struct {
	ID        string
	TokenID   string
	To        string
	Amount    string
	Timestamp int64
}

type TransferRequest struct {
	TokenID    string
	From       string
	To         string
	Amount     string
	PrivateKey string
}

type TransferResponse struct {
	ID        string
	TokenID   string
	From      string
	To        string
	Amount    string
	Timestamp int64
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
	ID        string
	TokenID   string
	From      string
	To        string
	Amount    string
	Timestamp int64
}

type ApproveRequest struct {
	TokenID    string
	Owner      string
	Spender    string
	Amount     string
	PrivateKey string
}

type ApproveResponse struct {
	ID        string
	TokenID   string
	Owner     string
	Spender   string
	Amount    string
	Timestamp int64
}

type BurnRequest struct {
	TokenID    string
	From       string
	Amount     string
	PrivateKey string
}

type BurnResponse struct {
	ID        string
	TokenID   string
	From      string
	Amount    string
	Timestamp int64
}

type BalanceRequest struct {
	TokenID string
	Owner   string
}

type BalanceResponse struct {
	TokenID string
	Owner   string
	Amount  string
}

type AllowanceRequest struct {
	TokenID string
	Owner   string
	Spender string
}

type AllowanceResponse struct {
	TokenID string
	Owner   string
	Spender string
	Amount  string
}

type HistoryRequest struct {
	TokenID string
	Owner   string
	Limit   int
}

type HistoryResponse struct {
	Transfers []TransferResponse
}

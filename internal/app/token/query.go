package token

import (
	"encoding/base64"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/token"
)

// TokenInfoResponse carries a token's metadata (the same shape CreateToken
// returns) for the read-only info query.
type TokenInfoResponse = CreateTokenResponse

// GetTokenInfoUseCase returns a token's metadata from its ID, closing the
// CLI-only gap (the CLI exposed `token info` but the app/REST layer had no
// read path for a token's name/symbol/supply/decimals/owner).
type GetTokenInfoUseCase struct {
	service token.Service
}

func NewGetTokenInfoUseCase(service token.Service) *GetTokenInfoUseCase {
	return &GetTokenInfoUseCase{service: service}
}

func (uc *GetTokenInfoUseCase) Execute(tokenID string) (*TokenInfoResponse, error) {
	t, err := uc.service.GetTokenInfo(token.TokenID(tokenID))
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, token.ErrTokenNotFound
	}
	return &TokenInfoResponse{
		ID:          string(t.ID()),
		Name:        t.Name(),
		Symbol:      t.Symbol(),
		TotalSupply: t.TotalSupply().String(),
		Decimals:    t.Decimals(),
		Owner:       base64.StdEncoding.EncodeToString(t.Owner()),
	}, nil
}

type GetBalanceUseCase struct {
	service token.Service
}

func NewGetBalanceUseCase(service token.Service) *GetBalanceUseCase {
	return &GetBalanceUseCase{service: service}
}

func (uc *GetBalanceUseCase) Execute(req *BalanceRequest) (*BalanceResponse, error) {
	owner, err := base64.StdEncoding.DecodeString(req.Owner)
	if err != nil {
		return nil, fmt.Errorf("invalid owner: %w", err)
	}

	amount, err := uc.service.GetBalance(token.TokenID(req.TokenID), owner)
	if err != nil {
		return nil, err
	}

	return &BalanceResponse{
		TokenID: req.TokenID,
		Owner:   req.Owner,
		Amount:  amount.String(),
	}, nil
}

type GetAllowanceUseCase struct {
	service token.Service
}

func NewGetAllowanceUseCase(service token.Service) *GetAllowanceUseCase {
	return &GetAllowanceUseCase{service: service}
}

func (uc *GetAllowanceUseCase) Execute(req *AllowanceRequest) (*AllowanceResponse, error) {
	owner, err := base64.StdEncoding.DecodeString(req.Owner)
	if err != nil {
		return nil, fmt.Errorf("invalid owner: %w", err)
	}

	spender, err := base64.StdEncoding.DecodeString(req.Spender)
	if err != nil {
		return nil, fmt.Errorf("invalid spender: %w", err)
	}

	amount, err := uc.service.GetAllowance(token.TokenID(req.TokenID), owner, spender)
	if err != nil {
		return nil, err
	}

	return &AllowanceResponse{
		TokenID: req.TokenID,
		Owner:   req.Owner,
		Spender: req.Spender,
		Amount:  amount.String(),
	}, nil
}

type GetHistoryUseCase struct {
	service token.Service
}

func NewGetHistoryUseCase(service token.Service) *GetHistoryUseCase {
	return &GetHistoryUseCase{service: service}
}

func (uc *GetHistoryUseCase) Execute(req *HistoryRequest) (*HistoryResponse, error) {
	owner, err := base64.StdEncoding.DecodeString(req.Owner)
	if err != nil {
		return nil, fmt.Errorf("invalid owner: %w", err)
	}

	events, err := uc.service.GetTransferHistory(token.TokenID(req.TokenID), owner, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}

	transfers := make([]TransferResponse, len(events))
	for i, e := range events {
		transfers[i] = TransferResponse{
			ID:          e.ID(),
			TokenID:     string(e.TokenID()),
			From:        base64.StdEncoding.EncodeToString(e.From()),
			To:          base64.StdEncoding.EncodeToString(e.To()),
			Amount:      e.Amount().String(),
			Timestamp:   e.Timestamp().Unix(),
			BlockHeight: e.BlockHeight(),
		}
	}

	return &HistoryResponse{
		Transfers: transfers,
	}, nil
}

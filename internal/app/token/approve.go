package token

import (
	"encoding/base64"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/token"
)

type ApproveUseCase struct {
	service token.Service
}

func NewApproveUseCase(service token.Service) *ApproveUseCase {
	return &ApproveUseCase{service: service}
}

func (uc *ApproveUseCase) Execute(req *ApproveRequest) (*ApproveResponse, error) {
	owner, err := base64.StdEncoding.DecodeString(req.Owner)
	if err != nil {
		return nil, fmt.Errorf("invalid owner: %w", err)
	}

	spender, err := base64.StdEncoding.DecodeString(req.Spender)
	if err != nil {
		return nil, fmt.Errorf("invalid spender: %w", err)
	}

	privKey, err := base64.StdEncoding.DecodeString(req.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	amount, err := token.NewAmountFromString(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	event, err := uc.service.Approve(&token.ApproveRequest{
		TokenID:    token.TokenID(req.TokenID),
		Owner:      owner,
		Spender:    spender,
		Amount:     amount,
		PrivateKey: privKey,
	})
	if err != nil {
		return nil, err
	}

	return &ApproveResponse{
		ID:        event.ID(),
		TokenID:   string(event.TokenID()),
		Owner:     base64.StdEncoding.EncodeToString(event.Owner()),
		Spender:   base64.StdEncoding.EncodeToString(event.Spender()),
		Amount:    event.Amount().String(),
		Timestamp: event.Timestamp().Unix(),
	}, nil
}

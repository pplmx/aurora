package token

import (
	"encoding/base64"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/token"
)

type TransferFromUseCase struct {
	service token.Service
}

func NewTransferFromUseCase(service token.Service) *TransferFromUseCase {
	return &TransferFromUseCase{service: service}
}

func (uc *TransferFromUseCase) Execute(req *TransferFromRequest) (*TransferFromResponse, error) {
	owner, err := base64.StdEncoding.DecodeString(req.Owner)
	if err != nil {
		return nil, fmt.Errorf("invalid owner: %w", err)
	}

	to, err := base64.StdEncoding.DecodeString(req.To)
	if err != nil {
		return nil, fmt.Errorf("invalid to: %w", err)
	}

	spender, err := base64.StdEncoding.DecodeString(req.Spender)
	if err != nil {
		return nil, fmt.Errorf("invalid spender: %w", err)
	}

	spenderKey, err := base64.StdEncoding.DecodeString(req.SpenderKey)
	if err != nil {
		return nil, fmt.Errorf("invalid spender key: %w", err)
	}

	amount, err := token.NewAmountFromString(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	event, err := uc.service.TransferFrom(&token.TransferFromRequest{
		TokenID:    token.TokenID(req.TokenID),
		Owner:      owner,
		To:         to,
		Amount:     amount,
		Spender:    spender,
		SpenderKey: spenderKey,
	})
	if err != nil {
		return nil, err
	}

	return &TransferFromResponse{
		ID:        event.ID(),
		TokenID:   string(event.TokenID()),
		From:      base64.StdEncoding.EncodeToString(event.From()),
		To:        base64.StdEncoding.EncodeToString(event.To()),
		Amount:    event.Amount().String(),
		Timestamp: event.Timestamp().Unix(),
	}, nil
}

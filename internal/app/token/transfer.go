package token

import (
	"encoding/base64"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/token"
)

type TransferUseCase struct {
	service token.Service
}

func NewTransferUseCase(service token.Service) *TransferUseCase {
	return &TransferUseCase{service: service}
}

func (uc *TransferUseCase) Execute(req *TransferRequest) (*TransferResponse, error) {
	from, err := base64.StdEncoding.DecodeString(req.From)
	if err != nil {
		return nil, fmt.Errorf("invalid from: %w", err)
	}

	to, err := base64.StdEncoding.DecodeString(req.To)
	if err != nil {
		return nil, fmt.Errorf("invalid to: %w", err)
	}

	privKey, err := base64.StdEncoding.DecodeString(req.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	amount, err := token.NewAmountFromString(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	event, err := uc.service.Transfer(&token.TransferRequest{
		TokenID:    token.TokenID(req.TokenID),
		From:       from,
		To:         to,
		Amount:     amount,
		PrivateKey: privKey,
	})
	if err != nil {
		return nil, err
	}

	return &TransferResponse{
		ID:        event.ID(),
		TokenID:   string(event.TokenID()),
		From:      base64.StdEncoding.EncodeToString(event.From()),
		To:        base64.StdEncoding.EncodeToString(event.To()),
		Amount:    event.Amount().String(),
		Timestamp: event.Timestamp().Unix(),
	}, nil
}

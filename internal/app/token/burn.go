package token

import (
	"encoding/base64"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/token"
)

type BurnUseCase struct {
	service token.Service
}

func NewBurnUseCase(service token.Service) *BurnUseCase {
	return &BurnUseCase{service: service}
}

func (uc *BurnUseCase) Execute(req *BurnRequest) (*BurnResponse, error) {
	from, err := base64.StdEncoding.DecodeString(req.From)
	if err != nil {
		return nil, fmt.Errorf("invalid from: %w", err)
	}

	privKey, err := base64.StdEncoding.DecodeString(req.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	amount, err := token.NewAmountFromString(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	event, err := uc.service.Burn(&token.BurnRequest{
		TokenID:    token.TokenID(req.TokenID),
		From:       from,
		Amount:     amount,
		PrivateKey: privKey,
	})
	if err != nil {
		return nil, err
	}

	return &BurnResponse{
		ID:        event.ID(),
		TokenID:   string(event.TokenID()),
		From:      base64.StdEncoding.EncodeToString(event.From()),
		Amount:    event.Amount().String(),
		Timestamp: event.Timestamp().Unix(),
	}, nil
}

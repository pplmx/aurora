package token

import (
	"encoding/base64"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/token"
)

type MintUseCase struct {
	service token.Service
}

func NewMintUseCase(service token.Service) *MintUseCase {
	return &MintUseCase{service: service}
}

func (uc *MintUseCase) Execute(req *MintRequest) (*MintResponse, error) {
	to, err := decodeKey("to", req.To)
	if err != nil {
		return nil, err
	}

	privKey, err := decodeKey("privateKey", req.PrivateKey)
	if err != nil {
		return nil, err
	}

	amount, err := token.NewAmountFromString(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	event, err := uc.service.Mint(&token.MintRequest{
		TokenID:    token.TokenID(req.TokenID),
		To:         to,
		Amount:     amount,
		PrivateKey: privKey,
	})
	if err != nil {
		return nil, err
	}

	return &MintResponse{
		ID:          event.ID(),
		TokenID:     string(event.TokenID()),
		To:          base64.StdEncoding.EncodeToString(event.To()),
		Amount:      event.Amount().String(),
		Timestamp:   event.Timestamp().Unix(),
		BlockHeight: event.BlockHeight(),
	}, nil
}

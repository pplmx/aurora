package tokenapp

import (
	"encoding/base64"

	"github.com/pplmx/aurora/internal/domain/token"
)

type TransferUseCase struct {
	service token.Service
}

func NewTransferUseCase(service token.Service) *TransferUseCase {
	return &TransferUseCase{service: service}
}

func (uc *TransferUseCase) Execute(req *TransferRequest) (*TransferResponse, error) {
	from, err := decodeKey("from", req.From)
	if err != nil {
		return nil, err
	}

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
		// The sentinel already reads "invalid amount"; repeating it as a prefix
		// produced "invalid amount: invalid amount" in the API and the CLI.
		return nil, err
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
		ID:          event.ID(),
		TokenID:     string(event.TokenID()),
		From:        base64.StdEncoding.EncodeToString(event.From()),
		To:          base64.StdEncoding.EncodeToString(event.To()),
		Amount:      event.Amount().String(),
		Timestamp:   event.Timestamp().Unix(),
		BlockHeight: event.BlockHeight(),
	}, nil
}

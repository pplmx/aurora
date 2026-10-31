package tokenapp

import (
	"encoding/base64"

	"github.com/pplmx/aurora/internal/domain/token"
)

type BurnUseCase struct {
	service token.Service
}

func NewBurnUseCase(service token.Service) *BurnUseCase {
	return &BurnUseCase{service: service}
}

func (uc *BurnUseCase) Execute(req *BurnRequest) (*BurnResponse, error) {
	from, err := decodeKey("from", req.From)
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
		ID:          event.ID(),
		TokenID:     string(event.TokenID()),
		From:        base64.StdEncoding.EncodeToString(event.From()),
		Amount:      event.Amount().String(),
		Timestamp:   event.Timestamp().Unix(),
		BlockHeight: event.BlockHeight(),
	}, nil
}

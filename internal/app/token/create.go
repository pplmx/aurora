package token

import (
	"encoding/base64"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/token"
)

type CreateTokenUseCase struct {
	service token.Service
}

func NewCreateTokenUseCase(service token.Service) *CreateTokenUseCase {
	return &CreateTokenUseCase{service: service}
}

func (uc *CreateTokenUseCase) Execute(req *CreateTokenRequest) (*CreateTokenResponse, error) {
	owner, err := base64.StdEncoding.DecodeString(req.Owner)
	if err != nil {
		return nil, fmt.Errorf("invalid owner: %w", err)
	}

	totalSupply, err := token.NewAmountFromString(req.TotalSupply)
	if err != nil {
		return nil, fmt.Errorf("invalid total supply: %w", err)
	}

	t, err := uc.service.CreateToken(&token.CreateTokenRequest{
		Name:        req.Name,
		Symbol:      req.Symbol,
		TotalSupply: totalSupply,
		Owner:       owner,
	})
	if err != nil {
		return nil, err
	}

	return &CreateTokenResponse{
		ID:          string(t.ID()),
		Name:        t.Name(),
		Symbol:      t.Symbol(),
		TotalSupply: t.TotalSupply().String(),
		Decimals:    t.Decimals(),
		Owner:       base64.StdEncoding.EncodeToString(t.Owner()),
	}, nil
}

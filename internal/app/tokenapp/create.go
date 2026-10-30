package tokenapp

import (
	"encoding/base64"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/token"
)

type Create struct {
	service token.Service
}

func NewCreate(service token.Service) *Create {
	return &Create{service: service}
}

func (uc *Create) Execute(req *CreateTokenRequest) (*CreateTokenResponse, error) {
	owner, err := decodeKey("owner", req.Owner)
	if err != nil {
		return nil, err
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
		Decimals:    req.Decimals,
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
		Mintable:    t.IsMintable(),
		Burnable:    t.IsBurnable(),
	}, nil
}

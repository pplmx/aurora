package deskapp

import (
	"github.com/pplmx/aurora/desktop/internal/nativeui"

	"github.com/pplmx/aurora/internal/app/tokenapp"
)

// --- Token ---------------------------------------------------------------

func (s *nativeServices) CreateToken(name, symbol, supply, owner string, decimals int) (nativeui.TokenRow, error) {
	uc := tokenapp.NewCreate(s.tokenService)
	d := int8(decimals)
	resp, err := uc.Execute(&tokenapp.CreateTokenRequest{
		Name: name, Symbol: symbol, TotalSupply: supply, Owner: owner, Decimals: &d,
	})
	if err != nil {
		return nativeui.TokenRow{}, err
	}
	return nativeui.TokenRow{
		ID: resp.ID, Name: resp.Name, Symbol: resp.Symbol,
		TotalSupply: resp.TotalSupply, Owner: resp.Owner,
	}, nil
}

func (s *nativeServices) TokenInfo(id string) (nativeui.TokenRow, error) {
	uc := tokenapp.NewGetTokenInfoUseCase(s.tokenService)
	resp, err := uc.Execute(id)
	if err != nil {
		return nativeui.TokenRow{}, err
	}
	return nativeui.TokenRow{
		ID: resp.ID, Name: resp.Name, Symbol: resp.Symbol,
		TotalSupply: resp.TotalSupply, Decimals: int(resp.Decimals), Owner: resp.Owner,
	}, nil
}

func (s *nativeServices) ListTokens(limit, offset int) ([]nativeui.TokenRow, error) {
	uc := tokenapp.NewListTokensUseCase(s.tokenService)
	resp, err := uc.Execute(&tokenapp.ListTokensRequest{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	rows := make([]nativeui.TokenRow, 0, len(resp))
	for _, t := range resp {
		rows = append(rows, nativeui.TokenRow{
			ID: t.ID, Name: t.Name, Symbol: t.Symbol,
			TotalSupply: t.TotalSupply, Decimals: int(t.Decimals), Owner: t.Owner,
		})
	}
	return rows, nil
}

func (s *nativeServices) TokenBalance(tokenID, owner string) (string, error) {
	uc := tokenapp.NewGetBalanceUseCase(s.tokenService)
	resp, err := uc.Execute(&tokenapp.BalanceRequest{TokenID: tokenID, Owner: owner})
	if err != nil {
		return "", err
	}
	return resp.Amount, nil
}

func (s *nativeServices) MintToken(tokenID, to, amount, priv string) error {
	uc := tokenapp.NewMintUseCase(s.tokenService)
	_, err := uc.Execute(&tokenapp.MintRequest{TokenID: tokenID, To: to, Amount: amount, PrivateKey: priv})
	return err
}

func (s *nativeServices) TransferToken(tokenID, from, to, amount, priv string) error {
	uc := tokenapp.NewTransferUseCase(s.tokenService)
	_, err := uc.Execute(&tokenapp.TransferRequest{
		TokenID: tokenID, From: from, To: to, Amount: amount, PrivateKey: priv,
	})
	return err
}

func (s *nativeServices) ApproveToken(tokenID, owner, spender, amount, priv string) error {
	uc := tokenapp.NewApproveUseCase(s.tokenService)
	_, err := uc.Execute(&tokenapp.ApproveRequest{
		TokenID: tokenID, Owner: owner, Spender: spender, Amount: amount, PrivateKey: priv,
	})
	return err
}

func (s *nativeServices) TransferFromToken(tokenID, owner, to, amount, spenderPriv string) error {
	uc := tokenapp.NewTransferFromUseCase(s.tokenService)
	_, err := uc.Execute(&tokenapp.TransferFromRequest{
		TokenID: tokenID, Owner: owner, To: to, Amount: amount, SpenderKey: spenderPriv,
	})
	return err
}

func (s *nativeServices) BurnToken(tokenID, from, amount, priv string) error {
	uc := tokenapp.NewBurnUseCase(s.tokenService)
	_, err := uc.Execute(&tokenapp.BurnRequest{
		TokenID: tokenID, From: from, Amount: amount, PrivateKey: priv,
	})
	return err
}

func (s *nativeServices) TokenAllowance(tokenID, owner, spender string) (string, error) {
	uc := tokenapp.NewGetAllowanceUseCase(s.tokenService)
	resp, err := uc.Execute(&tokenapp.AllowanceRequest{TokenID: tokenID, Owner: owner, Spender: spender})
	if err != nil {
		return "", err
	}
	return resp.Amount, nil
}

func (s *nativeServices) TokenHistory(tokenID, owner string, limit, offset int) ([]nativeui.TransferRow, error) {
	uc := tokenapp.NewGetHistoryUseCase(s.tokenService)
	resp, err := uc.Execute(&tokenapp.HistoryRequest{
		TokenID: tokenID, Owner: owner, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	rows := make([]nativeui.TransferRow, 0, len(resp.Transfers))
	for _, t := range resp.Transfers {
		rows = append(rows, transferRow(t))
	}
	return rows, nil
}

// transferRow maps one history entry into the view row. Nonce is part of the
// row the Token page renders; dropping it here made every transfer show
// nonce 0 even after the use case started returning it.
func transferRow(t tokenapp.TransferResponse) nativeui.TransferRow {
	return nativeui.TransferRow{From: t.From, To: t.To, Amount: t.Amount, Nonce: t.Nonce}
}

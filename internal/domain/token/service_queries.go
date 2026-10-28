package token

import (
	"database/sql"
)

func (s *TokenService) CreateToken(req *CreateTokenRequest) (*Token, error) {
	if err := ValidateTokenName(req.Name); err != nil {
		return nil, err
	}
	if err := ValidateTokenSymbol(req.Symbol); err != nil {
		return nil, err
	}
	if err := ValidateAmount(req.TotalSupply); err != nil {
		return nil, err
	}
	if err := ValidateOwner(req.Owner); err != nil {
		return nil, err
	}
	decimals := int8(defaultDecimals)
	if req.Decimals != nil {
		decimals = *req.Decimals
	}
	if err := ValidateTokenDecimals(decimals); err != nil {
		return nil, err
	}

	// A token's ID is its symbol (see NewToken). The persistence layer uses
	// INSERT OR REPLACE keyed on that ID, so creating a token with a symbol
	// that already exists would silently overwrite the existing token's row
	// and owner balance — losing the first token's data. Reject duplicates up
	// front so a create is either a genuine new token or an explicit error,
	// never a silent data-loss (TASK-075, ISS-067).
	tokenID := TokenID(req.Symbol)
	existing, err := s.repo.GetToken(tokenID)
	if err != nil && err != ErrTokenNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, ErrTokenExists
	}

	token := NewTokenWithDecimals(tokenID, req.Name, req.Symbol, req.TotalSupply, req.Owner, decimals)

	// CreateToken performs two writes (token row + owner balance) that must
	// be atomic: a failure between them would otherwise leave a token with
	// zero owner balance.
	err = s.txManager.WithTransaction(func(tx *sql.Tx) error {
		r := s.txRepo(tx)
		if err := r.SaveToken(token); err != nil {
			return err
		}
		return r.SetAccountBalance(token.ID(), req.Owner, req.TotalSupply)
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (s *TokenService) GetTokenInfo(tokenID TokenID) (*Token, error) {
	return s.repo.GetToken(tokenID)
}

// ListTokens returns stored tokens newest first, bounded by limit/offset.
func (s *TokenService) ListTokens(limit, offset int) ([]*Token, error) {
	return s.repo.ListTokens(limit, offset)
}

// requireToken returns ErrTokenNotFound unless a token with tokenID exists,
// mirroring the existence guard every mutator (Mint/Burn/Transfer/TransferFrom/
// Approve) applies before touching a row. The read paths (GetBalance/
// GetAllowance/GetTransferHistory) previously skipped it: a typo'd or never
// created token_id read back as a legitimate zero balance / empty history on
// the REST surface while /token/info 404'd — masking an existence probe as
// "no activity" instead of the unknown-resource->404 contract the rest of the
// platform enforces (ISS-250). Handles both not-found conventions the repos
// use (ErrTokenNotFound as err, or a nil token without error).
func (s *TokenService) requireToken(tokenID TokenID) error {
	t, err := s.repo.GetToken(tokenID)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrTokenNotFound
	}
	return nil
}

func (s *TokenService) GetBalance(tokenID TokenID, owner PublicKey) (*Amount, error) {
	if err := ValidatePublicKey(owner); err != nil {
		return nil, err
	}
	if err := s.requireToken(tokenID); err != nil {
		return nil, err
	}
	return s.repo.GetAccountBalance(tokenID, owner)
}

func (s *TokenService) GetAllowance(tokenID TokenID, owner, spender PublicKey) (*Amount, error) {
	if err := ValidatePublicKey(owner); err != nil {
		return nil, err
	}
	if err := ValidatePublicKey(spender); err != nil {
		return nil, err
	}
	if err := s.requireToken(tokenID); err != nil {
		return nil, err
	}
	approval, err := s.repo.GetApproval(tokenID, owner, spender)
	if err != nil {
		return nil, err
	}
	if approval == nil {
		return NewAmount(0), nil
	}
	return approval.Amount(), nil
}

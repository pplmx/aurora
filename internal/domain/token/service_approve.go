package token

import (
	"errors"
	"fmt"
	"math/big"
)

func (s *TokenService) Approve(req *ApproveRequest) (*ApproveEvent, error) {
	// Guard: refuse to approve an allowance for a token that does not
	// exist. Consistent with the other mutators (Mint/Burn/Transfer/
	// TransferFrom/Increase/DecreaseAllowance): without it a repo that
	// returns a nil token without error (e.g. a mock) would let SaveApproval
	// create an orphaned allowance row referencing a token that never existed.
	token, err := s.repo.GetToken(req.TokenID)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	if token == nil {
		return nil, ErrTokenNotFound
	}

	if err := VerifyPrivateKeyMatches(req.Owner, req.PrivateKey); err != nil {
		return nil, err
	}

	if err := ValidatePublicKey(req.Owner); err != nil {
		return nil, err
	}
	if err := ValidatePublicKey(req.Spender); err != nil {
		return nil, err
	}
	if err := ValidateAmount(req.Amount); err != nil {
		return nil, err
	}

	approval := NewApproval(req.TokenID, req.Owner, req.Spender, req.Amount)
	if err := s.repo.SaveApproval(approval); err != nil {
		return nil, err
	}

	event := NewApproveEvent(req.TokenID, req.Owner, req.Spender, req.Amount)
	if err := s.eventBus.Publish(event); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuditPublishFailed, err)
	}

	return event, nil
}

func (s *TokenService) IncreaseAllowance(req *AllowanceRequest) (*ApproveEvent, error) {
	// Guard: refuse to adjust allowance for a token that does not exist.
	// Every other token mutator (Approve/Mint/Burn/Transfer/TransferFrom)
	// verifies the token first; without this check TryAdjustApproval would
	// happily INSERT an allowance row for a dangling token_id, creating
	// orphaned approval data that references nothing.
	token, err := s.repo.GetToken(req.TokenID)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	if token == nil {
		return nil, ErrTokenNotFound
	}
	if err := VerifyPrivateKeyMatches(req.Owner, req.PrivateKey); err != nil {
		return nil, err
	}
	if err := ValidatePublicKey(req.Owner); err != nil {
		return nil, err
	}
	if err := ValidatePublicKey(req.Spender); err != nil {
		return nil, err
	}
	if err := ValidateAmount(req.Amount); err != nil {
		return nil, err
	}

	// Atomic primitive: closes the TOCTOU race where two concurrent
	// IncreaseAllowance(req, +10) calls both read allowance=50, both
	// compute 60, and both write 60, silently losing one increment.
	newAmount, err := s.repo.TryAdjustApproval(req.TokenID, req.Owner, req.Spender, req.Amount)
	if err != nil {
		return nil, err
	}

	event := NewApproveEvent(req.TokenID, req.Owner, req.Spender, newAmount)
	if err := s.eventBus.Publish(event); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuditPublishFailed, err)
	}
	return event, nil
}

func (s *TokenService) DecreaseAllowance(req *AllowanceRequest) (*ApproveEvent, error) {
	// Guard: see IncreaseAllowance — never adjust allowance for a token
	// that does not exist. Prevents orphaned allowance rows for a dangling
	// token_id.
	token, err := s.repo.GetToken(req.TokenID)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	if token == nil {
		return nil, ErrTokenNotFound
	}
	if err := VerifyPrivateKeyMatches(req.Owner, req.PrivateKey); err != nil {
		return nil, err
	}
	if err := ValidatePublicKey(req.Owner); err != nil {
		return nil, err
	}
	if err := ValidatePublicKey(req.Spender); err != nil {
		return nil, err
	}
	if err := ValidateAmount(req.Amount); err != nil {
		return nil, err
	}

	// Negate the amount to express "subtract this much" as a
	// signed delta for TryAdjustApproval. The primitive clamps at
	// zero, matching the pre-fix behaviour where newAmount.Sign()
	// < 0 was replaced with NewAmount(0).
	negDelta := &Amount{new(big.Int).Neg(req.Amount.Int)}

	// Atomic primitive: closes the TOCTOU race where two concurrent
	// DecreaseAllowance(req, -10) calls both read allowance=50, both
	// compute 40, and both write 40, silently losing one decrement.
	newAmount, err := s.repo.TryAdjustApproval(req.TokenID, req.Owner, req.Spender, negDelta)
	if err != nil {
		return nil, err
	}

	event := NewApproveEvent(req.TokenID, req.Owner, req.Spender, newAmount)
	if err := s.eventBus.Publish(event); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuditPublishFailed, err)
	}
	return event, nil
}

package token

import (
	"encoding/base64"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/errdetail"
)

func (s *TokenService) Mint(req *MintRequest) (*MintEvent, error) {
	token, err := s.repo.GetToken(req.TokenID)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, ErrTokenNotFound
	}

	if !token.IsMintable() {
		return nil, ErrTokenNotMintable
	}

	if err := VerifyPrivateKeyMatches(token.Owner(), req.PrivateKey); err != nil {
		return nil, err
	}

	if err := ValidateRecipient(req.To); err != nil {
		return nil, err
	}
	if err := ValidateAmount(req.Amount); err != nil {
		return nil, err
	}

	event := NewMintEvent(req.TokenID, req.To, req.Amount)

	data := fmt.Sprintf("mint|%s|%s", req.TokenID, base64.StdEncoding.EncodeToString(req.To))
	_, err = s.withBlockTransaction(data, func(r Repository, height int64) error {
		event.SetBlockHeight(height)

		// Atomic supply increment: closes the TOCTOU window
		// where two concurrent mints both read total_supply,
		// both added their amount in memory via AddToSupply,
		// and the last SaveToken clobbered the other mint's
		// increment — silently producing less total_supply
		// than the sum of all mints.
		if _, err := r.TryAddToSupply(req.TokenID, req.Amount); err != nil {
			return err
		}

		// Atomic add: closes the race where two concurrent Mints
		// to the same account both read currentBalance, compute
		// currentBalance + amount, and one overwrites the other.
		if _, err := r.TryAddBalance(req.TokenID, req.To, req.Amount); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// The audit event is published only AFTER the transaction commits
	// (ISS-074). eventBus writes to its own events store, which a token DB
	// rollback cannot undo, so publishing inside the transaction left a
	// phantom event behind whenever a later step rolled the mint back.
	if err := s.eventBus.Publish(event); err != nil {
		return nil, errdetail.Wrap(ErrAuditPublishFailed, err)
	}

	return event, nil
}

package token

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func (s *TokenService) Burn(req *BurnRequest) (*BurnEvent, error) {
	token, err := s.repo.GetToken(req.TokenID)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, ErrTokenNotFound
	}

	if !token.IsBurnable() {
		return nil, ErrTokenNotBurnable
	}

	if err := VerifyPrivateKeyMatches(req.From, req.PrivateKey); err != nil {
		return nil, err
	}

	if err := ValidatePublicKey(req.From); err != nil {
		return nil, err
	}
	if err := ValidateAmount(req.Amount); err != nil {
		return nil, err
	}

	event := NewBurnEvent(req.TokenID, req.From, req.Amount)

	data := fmt.Sprintf("burn|%s|%s", req.TokenID, req.From)
	_, err = s.withBlockTransaction(data, func(r Repository, height int64) error {
		event.SetBlockHeight(height)

		// Atomic balance subtract: closes the TOCTOU race the
		// pre-fix path had, where two concurrent burns both
		// read the same balance, both passed the Cmp(amount)
		// check, and both wrote back balance - amount,
		// silently allowing overdraw. The same primitive that
		// fixed Transfer/Mint/TransferFrom in Round 20 closes
		// this gap too.
		if _, err := r.TrySubtractBalance(req.TokenID, req.From, req.Amount); err != nil {
			return err
		}

		// Burning removes tokens from circulation, so it must also
		// decrement total_supply to preserve the ledger invariant
		// total_supply == sum of all balances. Both writes are in
		// the same transaction and roll back together.
		if _, err := r.TrySubtractFromSupply(req.TokenID, req.Amount); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Publish the audit event only after the transaction commits (ISS-074);
	// see the identical note in Mint.
	if err := s.eventBus.Publish(event); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuditPublishFailed, err)
	}

	return event, nil
}

func (s *TokenService) GetTransferHistory(tokenID TokenID, owner PublicKey, limit, offset int) ([]*TransferEvent, error) {
	if len(owner) > 0 {
		if err := ValidatePublicKey(owner); err != nil {
			return nil, err
		}
	}
	if err := s.requireToken(tokenID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = defaultHistoryLimit
	}
	var (
		events []*TransferEvent
		err    error
	)
	if len(owner) == 0 {
		events, err = s.eventReader.GetTransferEventsByToken(tokenID, limit, offset)
	} else {
		events, err = s.eventReader.GetTransferEventsByOwner(tokenID, owner, limit, offset)
	}
	if err != nil {
		return nil, err
	}

	return events, nil
}

func (s *TokenService) signMessage(tokenID TokenID, from, to PublicKey, amount *Amount, nonce uint64) []byte {
	msg := fmt.Sprintf("%s|%s|%s|%s|%d",
		tokenID,
		base64.StdEncoding.EncodeToString(from),
		base64.StdEncoding.EncodeToString(to),
		amount.String(),
		nonce,
	)
	hash := sha256.Sum256([]byte(msg))
	return hash[:]
}

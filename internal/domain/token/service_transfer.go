package token

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
)

func (s *TokenService) Transfer(req *TransferRequest) (*TransferEvent, error) {
	token, err := s.repo.GetToken(req.TokenID)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, ErrTokenNotFound
	}

	if err := ValidatePublicKey(req.From); err != nil {
		return nil, err
	}
	if err := ValidateRecipient(req.To); err != nil {
		return nil, err
	}
	if err := VerifyPrivateKeyMatches(req.From, req.PrivateKey); err != nil {
		return nil, err
	}
	if err := ValidateAmount(req.Amount); err != nil {
		return nil, err
	}

	// No balance pre-check here: the atomic TrySubtractBalance in the
	// transaction below is the single authority — a non-atomic read up front
	// ran against a different snapshot, so a concurrent top-up between the
	// read and the transaction returned a spurious INSUFFICIENT_BALANCE 400
	// even though the atomic debit would (and did) succeed (TASK-265,
	// ISS-261). The atomic primitive classifies the error itself, and the
	// handler maps its sentinel to the same 400 code.

	nonce, err := s.replay.ClaimNextNonce(string(req.TokenID), req.From)
	if err != nil {
		return nil, err
	}

	signature := ed25519.Sign(ed25519.PrivateKey(req.PrivateKey), s.signMessage(req.TokenID, req.From, req.To, req.Amount, nonce))

	event := NewTransferEvent(req.TokenID, req.From, req.To, req.Amount, nonce, signature)

	data := fmt.Sprintf("transfer|%s|%s|%s", req.TokenID,
		base64.StdEncoding.EncodeToString(req.From), base64.StdEncoding.EncodeToString(req.To))
	_, err = s.withBlockTransaction(data, func(r Repository, height int64) error {
		event.SetBlockHeight(height)

		// ClaimNextNonce above already persisted the nonce atomically on the
		// replay store's own connection, so re-saving it here would be a
		// no-op — and where that store shares the token DB file (the API
		// server wiring), this write through a SECOND connection would
		// deadlock against this transaction's write lock (BEGIN IMMEDIATE,
		// v1.70 / ISS-076): the tx would wait on the separate
		// SaveNonce while SaveNonce waits for the tx's lock, tripping the 5s
		// busy timeout into a spurious 500 (ISS-078).

		// Atomic subtract: closes the TOCTOU race where two
		// concurrent transfers both read fromBalance, both pass
		// the check, and both write back (fromBalance - amount).
		// The debit and credit below run in the same transaction:
		// if either fails, the whole transfer rolls back, so a
		// partial debit can never be left behind.
		if _, err := r.TrySubtractBalance(req.TokenID, req.From, req.Amount); err != nil {
			return err
		}

		// Atomic add: closes the symmetric race on the credit
		// side (two concurrent transfers to the same recipient
		// could both read toBalance and both write back
		// toBalance + amount, losing one transfer's credit).
		if _, err := r.TryAddBalance(req.TokenID, req.To, req.Amount); err != nil {
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

func (s *TokenService) TransferFrom(req *TransferFromRequest) (*TransferEvent, error) {
	token, err := s.repo.GetToken(req.TokenID)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, ErrTokenNotFound
	}

	if err := ValidatePublicKey(req.Owner); err != nil {
		return nil, err
	}
	if err := ValidateRecipient(req.To); err != nil {
		return nil, err
	}
	if err := ValidatePublicKey(req.Spender); err != nil {
		return nil, err
	}
	if err := VerifyPrivateKeyMatches(req.Spender, req.SpenderKey); err != nil {
		return nil, err
	}
	if err := ValidateAmount(req.Amount); err != nil {
		return nil, err
	}

	approval, err := s.repo.GetApproval(req.TokenID, req.Owner, req.Spender)
	if err != nil {
		return nil, err
	}
	if approval == nil {
		// A nonexistent allowance is distinguished up front so the caller
		// still gets INSUFFICIENT_ALLOWANCE rather than the repo's not-found
		// error; the atomic TryDeductApproval below is the authority on the
		// remaining amount checks (TASK-265, ISS-261).
		return nil, ErrInsufficientAllowance
	}

	// No balance/allowance pre-check here: the atomic TryDeductApproval and
	// TrySubtractBalance in the transaction below are the single authorities.
	// Non-atomic reads up front ran against a different snapshot, so a
	// concurrent allowance increase / owner top-up between the read and the
	// transaction returned a spurious 400 even though the atomic primitives
	// would (and did) succeed (TASK-265, ISS-261). Each primitive classifies
	// its own error and the handler maps the sentinels to the same codes.

	nonce, err := s.replay.ClaimNextNonce(string(req.TokenID), req.Spender)
	if err != nil {
		return nil, err
	}

	signature := ed25519.Sign(ed25519.PrivateKey(req.SpenderKey), s.signMessage(req.TokenID, req.Owner, req.To, req.Amount, nonce))

	event := NewTransferEvent(req.TokenID, req.Owner, req.To, req.Amount, nonce, signature)

	data := fmt.Sprintf("transferfrom|%s|%s|%s", req.TokenID,
		base64.StdEncoding.EncodeToString(req.Owner), base64.StdEncoding.EncodeToString(req.To))
	_, err = s.withBlockTransaction(data, func(r Repository, height int64) error {
		event.SetBlockHeight(height)

		// No replay.SaveNonce here for the same reason as Transfer: the
		// nonce was already persisted atomically by ClaimNextNonce above, and
		// writing it again via the replay store's separate connection would
		// deadlock against this transaction's write lock where both share the
		// same SQLite file (v1.70 / ISS-076, see Transfer).

		// All three mutations (allowance deduction, owner debit,
		// recipient credit) run in the same transaction. If any
		// step fails the transaction rolls back, so the allowance
		// and balances stay consistent — no hand-rolled
		// compensation is needed.
		//
		// Atomic allowance deduction: closes the TOCTOU race
		// where two concurrent TransferFroms both read
		// approval.Amount(), both pass the check, and both write
		// back approval - amount, allowing double-spend of the
		// allowance.
		if _, err := r.TryDeductApproval(req.TokenID, req.Owner, req.Spender, req.Amount); err != nil {
			return err
		}

		// Atomic balance subtract (owner) and add (recipient).
		if _, err := r.TrySubtractBalance(req.TokenID, req.Owner, req.Amount); err != nil {
			return err
		}
		if _, err := r.TryAddBalance(req.TokenID, req.To, req.Amount); err != nil {
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

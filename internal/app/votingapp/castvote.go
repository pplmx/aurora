// The ballot path: casting a signed vote as one transaction over the vote
// record, the session claim, the tally and the ledger block. The registration
// and session use cases live in their own files, so this one can be read as
// "what happens when a voter submits".
package votingapp

import (
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/pplmx/aurora/internal/domain/repoerr"
	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/pplmx/aurora/internal/ledger"
)

type CastVoteUseCase struct {
	repo      voting.TransactableRepository
	service   voting.Service
	txManager voting.TransactionManager
	chain     ledger.BlockWriter
}

// SetChain wires the blockchain writer used to record each accepted ballot
// on-chain. Optional and nil-safe: without it votes keep BlockHeight 0 (pure
// in-memory tests / fallback wiring). The voting package documents
// "blockchain-based vote recording", so the API server and CLI supply the
// chain.
func (uc *CastVoteUseCase) SetChain(chain ledger.BlockWriter) {
	uc.chain = chain
}

func NewCastVoteUseCase(repo voting.TransactableRepository, service voting.Service, txManager voting.TransactionManager) *CastVoteUseCase {
	if txManager == nil {
		txManager = noOpTxManager{}
	}
	return &CastVoteUseCase{
		repo:      repo,
		service:   service,
		txManager: txManager,
	}
}

// noOpTxManager executes the callback directly without a transaction, for
// callers over repositories without transaction support (in-memory fakes).
type noOpTxManager struct{}

func (noOpTxManager) WithTransaction(fn func(tx *sql.Tx) error) error {
	return fn(nil)
}

// NewCastVoteUseCaseWithoutTx builds the use case without a transaction
// manager. Only appropriate for repositories without transaction support;
// the SQLite-backed wiring must use NewCastVoteUseCase with a real
// TransactionManager.
func NewCastVoteUseCaseWithoutTx(repo voting.TransactableRepository, service voting.Service) *CastVoteUseCase {
	return NewCastVoteUseCase(repo, service, noOpTxManager{})
}

// txRepo returns the repository scoped to the given transaction when one is
// active, otherwise the use case's default repository. Every mutation inside
// the WithTransaction callback MUST go through txRepo so the voter claim,
// the vote row and the tally increment share one SQLite transaction and
// roll back together.
func (uc *CastVoteUseCase) txRepo(tx *sql.Tx) voting.Repository {
	if tx == nil {
		return uc.repo
	}
	return uc.repo.WithTx(tx)
}

func (uc *CastVoteUseCase) withVoteBlock(data string, fn func(repo voting.Repository, height int64) error) error {
	if uc.chain == nil {
		return uc.txManager.WithTransaction(func(tx *sql.Tx) error {
			return fn(uc.txRepo(tx), 0)
		})
	}

	if writer, ok := uc.chain.(ledger.BlockTransactionWriter); ok {
		return writer.WithBlockTransaction(data, uc.txManager, func(tx *sql.Tx, height int64) error {
			return fn(uc.txRepo(tx), height)
		})
	}

	height, err := uc.chain.AddBlock(data)
	if err != nil {
		return fmt.Errorf("failed to record vote on chain: %w", err)
	}
	return uc.txManager.WithTransaction(func(tx *sql.Tx) error {
		return fn(uc.txRepo(tx), height)
	})
}

func (uc *CastVoteUseCase) Execute(req CastVoteRequest) (*VoteResponse, error) {
	session, err := uc.repo.GetSession(req.SessionID)
	if err != nil {
		if errors.Is(err, repoerr.ErrNotFound) {
			// The SQLite repo signals a missing row with repoerr.ErrNotFound;
			// map it to the domain sentinel so the API returns 404 instead of
			// leaking an unclassified 500 (TASK-111, ISS-103).
			return nil, voting.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		return nil, voting.ErrSessionNotFound
	}

	// Both the explicit lifecycle state and the configured window gate every
	// ballot. In particular, a draft session within its nominal window still
	// requires the operator's start transition before it accepts votes.
	if err := session.CanAcceptVotes(time.Now()); err != nil {
		return nil, err
	}

	voter, err := uc.repo.GetVoter(req.VoterPublicKey)
	if err != nil {
		if errors.Is(err, repoerr.ErrNotFound) {
			return nil, voting.ErrVoterNotRegistered
		}
		return nil, fmt.Errorf("failed to get voter: %w", err)
	}
	if voter == nil {
		return nil, voting.ErrVoterNotRegistered
	}

	candidate, err := uc.repo.GetCandidate(req.CandidateID)
	if err != nil {
		if errors.Is(err, repoerr.ErrNotFound) {
			return nil, voting.ErrCandidateNotFound
		}
		return nil, fmt.Errorf("failed to get candidate: %w", err)
	}
	if candidate == nil {
		return nil, voting.ErrCandidateNotFound
	}

	// Ballot integrity: the candidate must belong to THIS session's roster.
	// Sessions declare their candidate set (session.Candidates), so a vote for
	// a registered-but-not-in-this-session candidate (possibly one in a
	// different election) must be rejected rather than inflating that
	// candidate's tally.
	if !containsString(session.Candidates, req.CandidateID) {
		return nil, voting.ErrCandidateNotInSession
	}

	privBytes, err := base64.StdEncoding.DecodeString(req.PrivateKey)
	if err != nil {
		// Client input error (HTTP 400), not a server fault: the raw stdlib
		// decode error previously escaped unclassified and came back as 500
		// INTERNAL_ERROR (TASK-095, ISS-089).
		return nil, fmt.Errorf("invalid private key encoding: %w", voting.ErrInvalidBase64)
	}
	// A valid-base64 but wrong-length key is the same client-error class, but
	// previously escaped as a 500: Ed25519Service.SignVote rejects the length
	// with a plain error the classifier does not match. Validate the length here
	// so the API can classify it as a 4xx, mirroring the NFT boundary
	// (internal/app/nft/decode.go) and ErrInvalidBase64 above (TASK-121, ISS-113).
	if len(privBytes) != ed25519.PrivateKeySize {
		return nil, voting.ErrInvalidPrivateKey
	}

	timestamp := time.Now().Unix()
	message := fmt.Sprintf("%s|%s|%d", req.VoterPublicKey, req.CandidateID, timestamp)
	signature, err := uc.service.SignVote(message, privBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to sign vote: %w", err)
	}
	// Ballot authenticity: the presented private key MUST correspond to the
	// registered voter's public key. If the signature does not verify against
	// the voter's stored public key, the caller does not own that voter's key
	// (they may know only the public key, which is public), so the ballot is
	// forged and must be rejected.
	if !uc.service.VerifyVote(voter.PublicKey, message, signature) {
		return nil, voting.ErrInvalidSignature
	}

	// The ballot checks and mutations run in ONE transaction. The
	// pre-transaction flow claimed the voter first and, on a later failure, ran
	// a best-effort UnmarkVoted/DeleteVote compensation whose own errors were
	// silently swallowed; a failed compensation permanently locked the voter
	// out or left an orphan vote row (violating the votes<->voters invariant).
	// With a real transaction no partial ballot state can ever commit.
	//
	// SaveVote inserts the (session_id, voter_pk) claim and the ballot in the
	// same transaction. The claim's primary key guarantees exactly one
	// concurrent caller for the same voter and session wins; the losers get
	// ErrAlreadyVoted and their whole transaction aborts with no side effects.
	var voteID string
	var blockHeight int64

	data := fmt.Sprintf("vote|%s|%s|%s", req.VoterPublicKey, req.CandidateID, req.SessionID)
	err = uc.withVoteBlock(data, func(repo voting.Repository, height int64) error {
		blockHeight = height

		// Re-read inside the transaction: an operator can end the session or
		// change its roster after the preflight check above. BEGIN IMMEDIATE
		// serializes this check with status updates, so a ballot cannot commit
		// against a session that was already ended in the write order.
		current, err := repo.GetSession(req.SessionID)
		if err != nil {
			if errors.Is(err, repoerr.ErrNotFound) {
				return voting.ErrSessionNotFound
			}
			return fmt.Errorf("failed to recheck session: %w", err)
		}
		if current == nil {
			return voting.ErrSessionNotFound
		}
		if err := current.CanAcceptVotes(time.Now()); err != nil {
			return err
		}
		if !containsString(current.Candidates, req.CandidateID) {
			return voting.ErrCandidateNotInSession
		}

		vote := voting.NewSessionVote(req.VoterPublicKey, req.CandidateID, req.SessionID, signature, message)
		vote.BlockHeight = height
		if err := repo.SaveVote(vote); err != nil {
			return fmt.Errorf("failed to save vote: %w", err)
		}
		voteID = vote.ID

		if err := repo.IncrementCandidateVoteCount(req.CandidateID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, voting.ErrAlreadyVoted) {
			return nil, voting.ErrAlreadyVoted
		}
		if errors.Is(err, repoerr.ErrNotFound) {
			// A row vanished between the existence checks above and the
			// transactional mutations: either the voter (the session claim
			// update) or the candidate (IncrementCandidateVoteCount) was deleted
			// concurrently. Re-check the voter to map to the right public
			// error.
			if v, vErr := uc.repo.GetVoter(req.VoterPublicKey); vErr == nil && v == nil {
				return nil, voting.ErrVoterNotRegistered
			}
			return nil, voting.ErrCandidateNotFound
		}
		return nil, fmt.Errorf("failed to cast vote: %w", err)
	}

	return &VoteResponse{
		ID:          voteID,
		SessionID:   req.SessionID,
		CandidateID: req.CandidateID,
		BlockHeight: blockHeight,
	}, nil
}

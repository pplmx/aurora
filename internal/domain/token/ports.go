package token

import (
	"github.com/pplmx/aurora/internal/domain/events"
)

// EventBus publishes a domain event to its subscribers. The token service
// declares the port here so it does not depend on an infrastructure package:
// an in-memory bus, a durable outbox or a test double all satisfy it.
type EventBus interface {
	Publish(events.Event) error
}

// ReplayProtection allocates monotonically increasing nonces per (token, owner)
// so two concurrent signed operations can never reuse one.
//
// ClaimNextNonce is the primitive that closes the TOCTOU race in Transfer and
// TransferFrom: a GetLastNonce/increment/SaveNonce sequence let two concurrent
// callers sign with the same nonce. Implementations MUST be concurrency-safe.
type ReplayProtection interface {
	GetLastNonce(tokenID string, owner []byte) (uint64, error)
	SaveNonce(tokenID string, owner []byte, nonce uint64) error
	ClaimNextNonce(tokenID string, owner []byte) (uint64, error)
}

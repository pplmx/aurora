package sqlite

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/pplmx/aurora/internal/domain/events"
	"github.com/stretchr/testify/require"
)

type mockEventStore struct {
	events []events.Event
	err    error
}

func (m *mockEventStore) Save(event events.Event) error                   { return nil }
func (m *mockEventStore) GetByType(string, int) ([]events.Event, error)   { return nil, nil }
func (m *mockEventStore) GetByModule(string, int) ([]events.Event, error) { return nil, nil }
func (m *mockEventStore) GetByAggregate(aggID string, limit, offset int) ([]events.Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	if limit <= 0 {
		limit = 50
	}
	if offset >= len(m.events) {
		return nil, nil
	}
	end := offset + limit
	if end > len(m.events) {
		end = len(m.events)
	}
	return m.events[offset:end], nil
}

func makeTransferEvent(id, tokenID, from, to string, amount string, nonce uint64, sig string) events.Event {
	payload, _ := json.Marshal(map[string]interface{}{
		"from":   from,
		"to":     to,
		"amount": amount,
		"nonce":  nonce,
		"sig":    sig,
	})
	return events.NewBaseEvent("token.transfer", tokenID, payload)
}

func makeMintEvent(id, tokenID, to string, amount string) events.Event {
	payload, _ := json.Marshal(map[string]interface{}{
		"to":     to,
		"amount": amount,
	})
	return events.NewBaseEvent("token.mint", tokenID, payload)
}

func makeBurnEvent(id, tokenID, from string, amount string) events.Event {
	payload, _ := json.Marshal(map[string]interface{}{
		"from":   from,
		"amount": amount,
	})
	return events.NewBaseEvent("token.burn", tokenID, payload)
}

func TestTokenEventReader_GetTransferEventsByOwner(t *testing.T) {
	owner := []byte("owner-key")
	recipient := []byte("recipient-key")
	ownerB64 := base64.StdEncoding.EncodeToString(owner)
	recipientB64 := base64.StdEncoding.EncodeToString(recipient)

	evt1 := makeTransferEvent("e1", "TOK", ownerB64, recipientB64, "100", 1, "sig1")
	evt2 := makeTransferEvent("e2", "TOK", recipientB64, ownerB64, "50", 2, "sig2")
	evt3 := makeMintEvent("e3", "TOK", ownerB64, "1000")

	store := &mockEventStore{events: []events.Event{evt1, evt2, evt3}}
	reader := NewTokenEventReader(store)

	results, err := reader.GetTransferEventsByOwner("TOK", owner, 50, 0)
	require.NoError(t, err)
	require.Len(t, results, 1, "only evt1 has owner 'from'")
	require.Equal(t, "TOK", string(results[0].TokenID()))
	require.Equal(t, "100", results[0].Amount().String())
}

func TestTokenEventReader_GetTransferEventsByOwner_Empty(t *testing.T) {
	store := &mockEventStore{events: nil}
	reader := NewTokenEventReader(store)

	results, err := reader.GetTransferEventsByOwner("TOK", []byte("owner"), 50, 0)
	require.NoError(t, err)
	require.Empty(t, results)
}

func TestTokenEventReader_GetTransferEventsByOwner_LimitAndOffset(t *testing.T) {
	owner := []byte("owner-key")
	ownerB64 := base64.StdEncoding.EncodeToString(owner)
	recipient := []byte("recipient-key")
	recipientB64 := base64.StdEncoding.EncodeToString(recipient)

	events := []events.Event{
		makeTransferEvent("e1", "TOK", ownerB64, recipientB64, "10", 1, "s1"),
		makeTransferEvent("e2", "TOK", ownerB64, recipientB64, "20", 2, "s2"),
		makeTransferEvent("e3", "TOK", ownerB64, recipientB64, "30", 3, "s3"),
	}

	store := &mockEventStore{events: events}
	reader := NewTokenEventReader(store)

	results, err := reader.GetTransferEventsByOwner("TOK", owner, 2, 1)
	require.NoError(t, err)
	require.Len(t, results, 2, "limit=2 offset=1 on 3 events returns 2 results")
}

func TestTokenEventReader_GetMintEventsByToken(t *testing.T) {
	to := []byte("recipient-key")
	toB64 := base64.StdEncoding.EncodeToString(to)

	evt1 := makeMintEvent("m1", "TOK", toB64, "500")
	evt2 := makeTransferEvent("t1", "TOK", toB64, toB64, "100", 1, "s1")
	evt3 := makeMintEvent("m2", "TOK", toB64, "600")

	store := &mockEventStore{events: []events.Event{evt1, evt2, evt3}}
	reader := NewTokenEventReader(store)

	results, err := reader.GetMintEventsByToken("TOK")
	require.NoError(t, err)
	require.Len(t, results, 2, "only mints should be returned")
	require.Equal(t, "500", results[0].Amount().String())
	require.Equal(t, "600", results[1].Amount().String())
}

func TestTokenEventReader_GetMintEventsByToken_Empty(t *testing.T) {
	store := &mockEventStore{events: nil}
	reader := NewTokenEventReader(store)

	results, err := reader.GetMintEventsByToken("TOK")
	require.NoError(t, err)
	require.Empty(t, results)
}

func TestTokenEventReader_GetBurnEventsByToken(t *testing.T) {
	from := []byte("owner-key")
	fromB64 := base64.StdEncoding.EncodeToString(from)
	recipientB64 := base64.StdEncoding.EncodeToString([]byte("recipient-key"))

	evt1 := makeBurnEvent("b1", "TOK", fromB64, "300")
	evt2 := makeTransferEvent("t1", "TOK", fromB64, recipientB64, "100", 1, "s1")
	evt3 := makeBurnEvent("b2", "TOK", fromB64, "200")

	store := &mockEventStore{events: []events.Event{evt1, evt2, evt3}}
	reader := NewTokenEventReader(store)

	results, err := reader.GetBurnEventsByToken("TOK")
	require.NoError(t, err)
	require.Len(t, results, 2, "only burns should be returned")
	require.Equal(t, "300", results[0].Amount().String())
	require.Equal(t, "200", results[1].Amount().String())
}

func TestTokenEventReader_GetBurnEventsByToken_Empty(t *testing.T) {
	store := &mockEventStore{events: nil}
	reader := NewTokenEventReader(store)

	results, err := reader.GetBurnEventsByToken("TOK")
	require.NoError(t, err)
	require.Empty(t, results)
}

var errTest = newTestError()

type testError struct{}

func (e *testError) Error() string { return "test error" }

func newTestError() error { return &testError{} }

func TestTokenEventReader_StoreError(t *testing.T) {
	store := &mockEventStore{err: errTest}
	reader := NewTokenEventReader(store)

	_, err := reader.GetTransferEventsByOwner("TOK", []byte("owner"), 10, 0)
	require.Error(t, err)
	require.ErrorIs(t, err, errTest)

	_, err = reader.GetMintEventsByToken("TOK")
	require.Error(t, err)

	_, err = reader.GetBurnEventsByToken("TOK")
	require.Error(t, err)
}

func TestTokenEventReader_SkipsMalformedPayload(t *testing.T) {
	owner := []byte("owner-key")
	ownerB64 := base64.StdEncoding.EncodeToString(owner)

	evt1 := events.NewBaseEvent("token.transfer", "TOK", []byte("not-valid-json"))
	evt2 := makeTransferEvent("e2", "TOK", ownerB64, ownerB64, "100", 1, "s1")

	store := &mockEventStore{events: []events.Event{evt1, evt2}}
	reader := NewTokenEventReader(store)

	results, err := reader.GetTransferEventsByOwner("TOK", owner, 50, 0)
	require.NoError(t, err)
	require.Len(t, results, 1, "malformed event should be skipped")
}

func TestTokenEventReader_Close(t *testing.T) {
	reader := NewTokenEventReader(&mockEventStore{})
	require.NoError(t, reader.Close())
}

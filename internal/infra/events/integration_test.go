package events

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/pplmx/aurora/internal/domain/events"
)

func TestFullFlow_CompositeBus(t *testing.T) {
	storeFile, err := os.CreateTemp("", "events-integration-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(storeFile.Name())
	storeFile.Close()

	eventStore, err := NewSQLiteEventStore(storeFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer eventStore.Close()

	bus := NewCompositeEventBus()
	bus.SubscribeAll(NewAuditHandler(eventStore).Handle)

	for i := 0; i < 5; i++ {
		payload, _ := json.Marshal(map[string]interface{}{"index": i})
		e := events.NewBaseEvent("test.action", "agg-1", payload)
		if err := bus.Publish(e); err != nil {
			t.Fatalf("Publish() error = %v", err)
		}
	}

	eventsResult, err := eventStore.GetByType("test.action", 10)
	if err != nil {
		t.Fatalf("GetByType() error = %v", err)
	}
	if len(eventsResult) != 5 {
		t.Errorf("len(events) = %d, want 5", len(eventsResult))
	}
}

func TestFullFlow_PublishRetrieve(t *testing.T) {
	storeFile, err := os.CreateTemp("", "events-integration-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(storeFile.Name())
	storeFile.Close()

	eventStore, err := NewSQLiteEventStore(storeFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer eventStore.Close()

	bus := NewCompositeEventBus()
	bus.SubscribeAll(NewAuditHandler(eventStore).Handle)

	testPayload := map[string]interface{}{
		"token_id": "token-123",
		"owner":    "owner-456",
		"amount":   100,
	}
	payloadBytes, _ := json.Marshal(testPayload)
	e := events.NewBaseEvent("token.mint", "token-123", payloadBytes)

	if err := bus.Publish(e); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	byType, err := eventStore.GetByType("token.mint", 10)
	if err != nil {
		t.Fatalf("GetByType() error = %v", err)
	}
	if len(byType) != 1 {
		t.Errorf("len(byType) = %d, want 1", len(byType))
	}

	byAgg, err := eventStore.GetByAggregate("token-123")
	if err != nil {
		t.Fatalf("GetByAggregate() error = %v", err)
	}
	if len(byAgg) != 1 {
		t.Errorf("len(byAgg) = %d, want 1", len(byAgg))
	}

	var parsed map[string]interface{}
	if err := events.ParsePayload(byType[0], &parsed); err != nil {
		t.Fatalf("ParsePayload() error = %v", err)
	}
	if parsed["token_id"] != "token-123" {
		t.Errorf("token_id = %v, want token-123", parsed["token_id"])
	}
}

func TestFullFlow_MultipleAggregates(t *testing.T) {
	storeFile, err := os.CreateTemp("", "events-integration-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(storeFile.Name())
	storeFile.Close()

	eventStore, err := NewSQLiteEventStore(storeFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer eventStore.Close()

	bus := NewCompositeEventBus()
	bus.SubscribeAll(NewAuditHandler(eventStore).Handle)

	for i := 0; i < 3; i++ {
		payload, _ := json.Marshal(map[string]interface{}{"index": i})
		e := events.NewBaseEvent("nft.transfer", "nft-"+string(rune('A'+i)), payload)
		if err := bus.Publish(e); err != nil {
			t.Fatalf("Publish() error = %v", err)
		}
	}

	eventsResult, err := eventStore.GetByType("nft.transfer", 10)
	if err != nil {
		t.Fatalf("GetByType() error = %v", err)
	}
	if len(eventsResult) != 3 {
		t.Errorf("len(events) = %d, want 3", len(eventsResult))
	}
}

func TestFullFlow_ReplayProtection_NonceManagement(t *testing.T) {
	storeFile, err := os.CreateTemp("", "replay-integration-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(storeFile.Name())
	storeFile.Close()

	replayProt, err := NewSQLiteReplayProtection(storeFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer replayProt.Close()

	tokenID := "token-abc"
	owner := []byte("owner-123")

	nonce, err := replayProt.GetLastNonce(tokenID, owner)
	if err != nil {
		t.Fatalf("GetLastNonce() error = %v", err)
	}
	if nonce != 0 {
		t.Errorf("initial nonce = %d, want 0", nonce)
	}

	if err := replayProt.SaveNonce(tokenID, owner, 1); err != nil {
		t.Fatalf("SaveNonce() error = %v", err)
	}

	nonce, err = replayProt.GetLastNonce(tokenID, owner)
	if err != nil {
		t.Fatalf("GetLastNonce() error = %v", err)
	}
	if nonce != 1 {
		t.Errorf("nonce after save = %d, want 1", nonce)
	}

	if err := replayProt.SaveNonce(tokenID, owner, 5); err != nil {
		t.Fatalf("SaveNonce() error = %v", err)
	}

	nonce, err = replayProt.GetLastNonce(tokenID, owner)
	if err != nil {
		t.Fatalf("GetLastNonce() error = %v", err)
	}
	if nonce != 5 {
		t.Errorf("nonce after update = %d, want 5", nonce)
	}
}

func TestFullFlow_ReplayProtection_MultipleOwners(t *testing.T) {
	storeFile, err := os.CreateTemp("", "replay-integration-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(storeFile.Name())
	storeFile.Close()

	replayProt, err := NewSQLiteReplayProtection(storeFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer replayProt.Close()

	tokenID := "token-multi"

	owner1 := []byte("alice")
	owner2 := []byte("bob")

	if err := replayProt.SaveNonce(tokenID, owner1, 10); err != nil {
		t.Fatalf("SaveNonce() for owner1 error = %v", err)
	}
	if err := replayProt.SaveNonce(tokenID, owner2, 20); err != nil {
		t.Fatalf("SaveNonce() for owner2 error = %v", err)
	}

	nonce1, err := replayProt.GetLastNonce(tokenID, owner1)
	if err != nil {
		t.Fatalf("GetLastNonce() for owner1 error = %v", err)
	}
	if nonce1 != 10 {
		t.Errorf("nonce1 = %d, want 10", nonce1)
	}

	nonce2, err := replayProt.GetLastNonce(tokenID, owner2)
	if err != nil {
		t.Fatalf("GetLastNonce() for owner2 error = %v", err)
	}
	if nonce2 != 20 {
		t.Errorf("nonce2 = %d, want 20", nonce2)
	}
}

func TestFullFlow_ReplayProtection_MultipleTokens(t *testing.T) {
	storeFile, err := os.CreateTemp("", "replay-integration-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(storeFile.Name())
	storeFile.Close()

	replayProt, err := NewSQLiteReplayProtection(storeFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer replayProt.Close()

	owner := []byte("owner-xyz")

	if err := replayProt.SaveNonce("token-A", owner, 1); err != nil {
		t.Fatalf("SaveNonce() for token-A error = %v", err)
	}
	if err := replayProt.SaveNonce("token-B", owner, 2); err != nil {
		t.Fatalf("SaveNonce() for token-B error = %v", err)
	}

	nonceA, err := replayProt.GetLastNonce("token-A", owner)
	if err != nil {
		t.Fatalf("GetLastNonce() for token-A error = %v", err)
	}
	if nonceA != 1 {
		t.Errorf("nonceA = %d, want 1", nonceA)
	}

	nonceB, err := replayProt.GetLastNonce("token-B", owner)
	if err != nil {
		t.Fatalf("GetLastNonce() for token-B error = %v", err)
	}
	if nonceB != 2 {
		t.Errorf("nonceB = %d, want 2", nonceB)
	}
}

func TestFullFlow_EventBusWithStatsHandler(t *testing.T) {
	storeFile, err := os.CreateTemp("", "events-integration-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(storeFile.Name())
	storeFile.Close()

	eventStore, err := NewSQLiteEventStore(storeFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer eventStore.Close()

	bus := NewCompositeEventBus()

	statsHandler := NewStatsHandler()
	auditHandler := NewAuditHandler(eventStore)

	bus.Subscribe("token.mint", statsHandler.Handle)
	bus.Subscribe("token.transfer", statsHandler.Handle)
	bus.SubscribeAll(auditHandler.Handle)

	for i := 0; i < 3; i++ {
		payload, _ := json.Marshal(map[string]interface{}{"index": i})
		e := events.NewBaseEvent("token.mint", "agg-1", payload)
		if err := bus.Publish(e); err != nil {
			t.Fatalf("Publish() error = %v", err)
		}
	}

	for i := 0; i < 2; i++ {
		payload, _ := json.Marshal(map[string]interface{}{"index": i})
		e := events.NewBaseEvent("token.transfer", "agg-2", payload)
		if err := bus.Publish(e); err != nil {
			t.Fatalf("Publish() error = %v", err)
		}
	}

	if count := statsHandler.GetCount("token.mint"); count != 3 {
		t.Errorf("token.mint count = %d, want 3", count)
	}
	if count := statsHandler.GetCount("token.transfer"); count != 2 {
		t.Errorf("token.transfer count = %d, want 2", count)
	}

	eventsResult, err := eventStore.GetByType("token.mint", 10)
	if err != nil {
		t.Fatalf("GetByType() error = %v", err)
	}
	if len(eventsResult) != 3 {
		t.Errorf("len(events) = %d, want 3", len(eventsResult))
	}
}

func TestFullFlow_Unsubscribe(t *testing.T) {
	storeFile, err := os.CreateTemp("", "events-integration-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(storeFile.Name())
	storeFile.Close()

	eventStore, err := NewSQLiteEventStore(storeFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer eventStore.Close()

	bus := NewCompositeEventBus()

	statsHandler := NewStatsHandler()
	unsubscribe := bus.Subscribe("test.event", statsHandler.Handle)

	payload1, _ := json.Marshal(map[string]interface{}{"index": 0})
	e1 := events.NewBaseEvent("test.event", "agg-1", payload1)
	if err := bus.Publish(e1); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	unsubscribe()

	payload2, _ := json.Marshal(map[string]interface{}{"index": 1})
	e2 := events.NewBaseEvent("test.event", "agg-1", payload2)
	if err := bus.Publish(e2); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	if count := statsHandler.GetCount("test.event"); count != 1 {
		t.Errorf("count after unsubscribe = %d, want 1", count)
	}

	_ = eventStore
}

func TestFullFlow_ModuleFiltering(t *testing.T) {
	storeFile, err := os.CreateTemp("", "events-integration-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(storeFile.Name())
	storeFile.Close()

	eventStore, err := NewSQLiteEventStore(storeFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer eventStore.Close()

	bus := NewCompositeEventBus()
	bus.SubscribeAll(NewAuditHandler(eventStore).Handle)

	bus.Publish(events.NewBaseEvent("token.mint", "agg-1", []byte(`{}`)))
	bus.Publish(events.NewBaseEvent("nft.mint", "agg-2", []byte(`{}`)))
	bus.Publish(events.NewBaseEvent("token.transfer", "agg-3", []byte(`{}`)))

	tokenEvents, err := eventStore.GetByModule("token", 10)
	if err != nil {
		t.Fatalf("GetByModule() error = %v", err)
	}
	if len(tokenEvents) != 2 {
		t.Errorf("len(tokenEvents) = %d, want 2", len(tokenEvents))
	}

	nftEvents, err := eventStore.GetByModule("nft", 10)
	if err != nil {
		t.Fatalf("GetByModule() error = %v", err)
	}
	if len(nftEvents) != 1 {
		t.Errorf("len(nftEvents) = %d, want 1", len(nftEvents))
	}
}

func TestFullFlow_CompositeBus_AllBuses(t *testing.T) {
	storeFile, err := os.CreateTemp("", "events-integration-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(storeFile.Name())
	storeFile.Close()

	eventStore, err := NewSQLiteEventStore(storeFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer eventStore.Close()

	bus := NewCompositeEventBus()

	statsHandler := NewStatsHandler()
	bus.AsyncBus.Subscribe("test.event", statsHandler.Handle)

	bus.SubscribeAll(NewAuditHandler(eventStore).Handle)

	e := events.NewBaseEvent("test.event", "agg-1", []byte(`{}`))
	if err := bus.Publish(e); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	eventsResult, err := eventStore.GetByType("test.event", 10)
	if err != nil {
		t.Fatalf("GetByType() error = %v", err)
	}
	if len(eventsResult) != 1 {
		t.Errorf("len(events) = %d, want 1", len(eventsResult))
	}
}
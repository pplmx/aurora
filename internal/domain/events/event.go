package events

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Event interface {
	ID() string
	EventType() string
	Module() string
	AggregateID() string
	Timestamp() time.Time
	Payload() []byte
}

type BaseEvent struct {
	id        string
	eventType string
	module    string
	aggID     string
	timestamp time.Time
	payload   []byte
}

func NewBaseEvent(eventType, aggID string, payload []byte) *BaseEvent {
	module := strings.SplitN(eventType, ".", 2)[0]
	return &BaseEvent{
		id:        uuid.New().String(),
		eventType: eventType,
		module:    module,
		aggID:     aggID,
		timestamp: time.Now(),
		payload:   payload,
	}
}

func (e *BaseEvent) ID() string           { return e.id }
func (e *BaseEvent) EventType() string    { return e.eventType }
func (e *BaseEvent) Module() string       { return e.module }
func (e *BaseEvent) AggregateID() string  { return e.aggID }
func (e *BaseEvent) Timestamp() time.Time { return e.timestamp }
func (e *BaseEvent) Payload() []byte      { return e.payload }

func ParsePayload(e Event, dest interface{}) error {
	return json.Unmarshal(e.Payload(), dest)
}

package events

import "errors"

var (
	ErrInvalidPayload = errors.New("invalid event payload")
	ErrEventNotFound  = errors.New("event not found")
	ErrHandlerFailed  = errors.New("handler failed")
	ErrEventBusClosed = errors.New("event bus is closed")
	ErrEventBusFull   = errors.New("event bus buffer full")
)

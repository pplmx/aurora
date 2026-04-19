package events

import (
	"encoding/json"
	"fmt"
)

type LotteryCreatedEvent struct {
	*BaseEvent
}

type lotteryCreatedPayload struct {
	Participants string `json:"participants"`
	WinnerCount  int    `json:"winner_count"`
}

func (e *LotteryCreatedEvent) Participants() (string, error) {
	var p lotteryCreatedPayload
	if err := json.Unmarshal(e.Payload(), &p); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return p.Participants, nil
}

func (e *LotteryCreatedEvent) WinnerCount() (int, error) {
	var p lotteryCreatedPayload
	if err := json.Unmarshal(e.Payload(), &p); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return p.WinnerCount, nil
}

type LotteryDrawnEvent struct {
	*BaseEvent
}

type lotteryDrawnPayload struct {
	Winners string `json:"winners"`
	Proof   string `json:"proof"`
}

func (e *LotteryDrawnEvent) Winners() (string, error) {
	var p lotteryDrawnPayload
	if err := json.Unmarshal(e.Payload(), &p); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return p.Winners, nil
}

func (e *LotteryDrawnEvent) Proof() (string, error) {
	var p lotteryDrawnPayload
	if err := json.Unmarshal(e.Payload(), &p); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return p.Proof, nil
}

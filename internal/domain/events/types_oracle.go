package events

import (
	"encoding/json"
	"fmt"
)

type OracleDataFetchedEvent struct {
	*BaseEvent
}

type oracleDataFetchedPayload struct {
	Source string      `json:"source"`
	Data   interface{} `json:"data"`
}

func (e *OracleDataFetchedEvent) Source() (string, error) {
	var p oracleDataFetchedPayload
	if err := json.Unmarshal(e.Payload(), &p); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return p.Source, nil
}

func (e *OracleDataFetchedEvent) Data() (interface{}, error) {
	var p oracleDataFetchedPayload
	if err := json.Unmarshal(e.Payload(), &p); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return p.Data, nil
}

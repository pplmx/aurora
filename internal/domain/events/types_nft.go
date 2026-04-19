package events

import (
	"encoding/json"
	"fmt"
)

type NFTMintEvent struct {
	*BaseEvent
}

type nftMintPayload struct {
	Metadata string `json:"metadata"`
}

func (e *NFTMintEvent) Owner() ([]byte, error) {
	return base64DecodeField(e.Payload(), "owner")
}

func (e *NFTMintEvent) Metadata() (string, error) {
	var p nftMintPayload
	if err := json.Unmarshal(e.Payload(), &p); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return p.Metadata, nil
}

type NFTTransferEvent struct {
	*BaseEvent
}

type nftTransferPayload struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (e *NFTTransferEvent) From() ([]byte, error) {
	return base64DecodeField(e.Payload(), "from")
}

func (e *NFTTransferEvent) To() ([]byte, error) {
	return base64DecodeField(e.Payload(), "to")
}

type NFTBurnEvent struct {
	*BaseEvent
}

type nftBurnPayload struct {
	From string `json:"from"`
}

func (e *NFTBurnEvent) From() ([]byte, error) {
	return base64DecodeField(e.Payload(), "from")
}

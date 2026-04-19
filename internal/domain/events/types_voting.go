package events

import "encoding/json"

type VotingCreatedEvent struct {
	*BaseEvent
}

type votingCreatedPayload struct {
	Proposal string `json:"proposal"`
}

func (e *VotingCreatedEvent) Proposer() ([]byte, error) {
	return base64DecodeField(e.Payload(), "proposer")
}

func (e *VotingCreatedEvent) Proposal() (string, error) {
	var p votingCreatedPayload
	if err := json.Unmarshal(e.Payload(), &p); err != nil {
		return "", err
	}
	return p.Proposal, nil
}

type VotingVoteEvent struct {
	*BaseEvent
}

type votingVotePayload struct {
	Choice string `json:"choice"`
}

func (e *VotingVoteEvent) Voter() ([]byte, error) {
	return base64DecodeField(e.Payload(), "voter")
}

func (e *VotingVoteEvent) Choice() (string, error) {
	var p votingVotePayload
	if err := json.Unmarshal(e.Payload(), &p); err != nil {
		return "", err
	}
	return p.Choice, nil
}

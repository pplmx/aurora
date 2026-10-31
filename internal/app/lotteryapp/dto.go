// Package lotteryapp holds the lottery use cases: create a draw, verify its VRF
// proof, and import or export history.
//
// It orchestrates the domain entity and the ledger; it never touches SQL or
// HTTP, which is what lets the CLI, the REST handler and the window share one
// implementation of a draw.
package lotteryapp

type CreateLotteryRequest struct {
	Participants string
	Seed         string
	WinnerCount  int
}

type LotteryResponse struct {
	ID              string   `json:"id"`
	BlockHeight     int64    `json:"block_height"`
	Seed            string   `json:"seed"`
	Participants    []string `json:"participants"`
	Winners         []string `json:"winners"`
	WinnerAddresses []string `json:"winner_addresses"`
	VRFProof        string   `json:"vrf_proof"`
	VRFOutput       string   `json:"vrf_output"`
	Timestamp       int64    `json:"timestamp"`
	Verified        bool     `json:"verified"`
}

package nativeui

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWriteHandlersRejectIncompleteForms covers the defensive half of the
// readiness contract. The primary buttons are disabled in the normal path;
// these direct calls represent a stale frame or an in-process caller that
// reaches the handler anyway.
func TestWriteHandlersRejectIncompleteForms(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*App)
		run     func(*App)
		status  func(*App) statusLine
		want    string
	}{
		{
			name: "create token",
			prepare: func(a *App) {
				a.token.name, a.token.symbol, a.token.supply = "Gold", "GLD", "100"
			},
			run:    func(a *App) { a.createToken() },
			status: func(a *App) statusLine { return a.token.status },
			want:   "Token name, symbol, supply, and owner are required.",
		},
		{
			name: "mint token",
			prepare: func(a *App) {
				a.token.tokenID, a.token.mintTo, a.token.mintAmount = "t1", "recipient", "5"
			},
			run:    func(a *App) { a.mintToken() },
			status: func(a *App) statusLine { return a.token.status },
			want:   "Token ID, recipient, amount, and private key are required.",
		},
		{
			name: "transfer token",
			prepare: func(a *App) {
				a.token.tokenID, a.token.xFrom, a.token.xTo, a.token.xAmount = "t1", "sender", "recipient", "5"
			},
			run:    func(a *App) { a.transferToken() },
			status: func(a *App) statusLine { return a.token.status },
			want:   "Token ID, sender, recipient, amount, and private key are required.",
		},
		{
			name: "approve token",
			prepare: func(a *App) {
				a.token.tokenID, a.token.approveOwner, a.token.approveTo, a.token.approveAmt = "t1", "owner", "spender", "5"
			},
			run:    func(a *App) { a.approveToken() },
			status: func(a *App) statusLine { return a.token.status },
			want:   "Token ID, owner, spender, amount, and private key are required.",
		},
		{
			name: "transfer from token",
			prepare: func(a *App) {
				a.token.tokenID, a.token.fromOwner, a.token.fromTo, a.token.fromAmount = "t1", "owner", "recipient", "5"
			},
			run:    func(a *App) { a.transferFromToken() },
			status: func(a *App) statusLine { return a.token.status },
			want:   "Token ID, owner, recipient, amount, and spender private key are required.",
		},
		{
			name: "mint NFT",
			prepare: func(a *App) {
				a.nft.name = "Star"
			},
			run:    func(a *App) { a.mintNFT() },
			status: func(a *App) statusLine { return a.nft.status },
			want:   "NFT name and creator are required.",
		},
		{
			name: "transfer NFT",
			prepare: func(a *App) {
				a.nft.transferID, a.nft.from, a.nft.to = "n1", "sender", "recipient"
			},
			run:    func(a *App) { a.transferNFT() },
			status: func(a *App) statusLine { return a.nft.status },
			want:   "NFT ID, sender, recipient, and private key are required.",
		},
		{
			name: "add source",
			prepare: func(a *App) {
				a.oracle.addName = "price"
			},
			run:    func(a *App) { a.addOracleSource() },
			status: func(a *App) statusLine { return a.oracle.status },
			want:   "Source name and URL are required.",
		},
		{
			name:   "create session",
			run:    func(a *App) { a.createVotingSession() },
			status: func(a *App) statusLine { return a.voting.sessionResult },
			want:   "Session title is required.",
		},
		{
			name: "cast vote",
			prepare: func(a *App) {
				a.voting.voteSession, a.voting.voteCandidate = "s1", "c1"
			},
			run:    func(a *App) { a.castVote() },
			status: func(a *App) statusLine { return a.voting.voteResult },
			want:   "Voter public key and private key are required.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := New(&fakeServices{
				candidates: []CandidateRow{{ID: "c1", Name: "Alice"}},
			})
			if tt.prepare != nil {
				tt.prepare(app)
			}

			tt.run(app)
			got := tt.status(app)
			require.True(t, got.isErr, "incomplete %s must fail before the service call", tt.name)
			require.Contains(t, got.text, tt.want)
		})
	}
}

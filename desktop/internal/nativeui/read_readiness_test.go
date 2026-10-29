package nativeui

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestReadHandlersRejectIncompleteForms covers the defensive half of the read
// readiness contract. Buttons are disabled in the normal path; these calls
// represent a stale frame or an in-process caller that reaches the handler
// anyway and must not turn an empty input into a service query.
func TestReadHandlersRejectIncompleteForms(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*App)
		run     func(*App)
		status  func(*App) statusLine
		calls   func(*fakeServices) int
		want    string
	}{
		{
			name:   "token info",
			run:    func(a *App) { a.readTokenInfo() },
			status: func(a *App) statusLine { return a.token.status },
			calls:  func(f *fakeServices) int { return f.tokenInfoCalls },
			want:   "Token ID is required for this read.",
		},
		{
			name: "token balance",
			prepare: func(a *App) {
				a.token.tokenID = "t1"
			},
			run:    func(a *App) { a.readTokenBalance() },
			status: func(a *App) statusLine { return a.token.status },
			calls:  func(f *fakeServices) int { return f.tokenBalanceCalls },
			want:   "Token ID and account are required for this read.",
		},
		{
			name: "token allowance",
			prepare: func(a *App) {
				a.token.tokenID, a.token.allowanceOwner = "t1", "owner"
			},
			run:    func(a *App) { a.loadAllowance() },
			status: func(a *App) statusLine { return a.token.status },
			calls:  func(f *fakeServices) int { return f.tokenAllowanceCalls },
			want:   "Token ID, owner, and spender are required for this read.",
		},
		{
			name:   "token history",
			run:    func(a *App) { a.readTokenHistory() },
			status: func(a *App) statusLine { return a.token.status },
			calls:  func(f *fakeServices) int { return f.tokenHistoryCalls },
			want:   "Token ID is required for this read.",
		},
		{
			name:   "NFT details",
			run:    func(a *App) { a.readNFTDetails() },
			status: func(a *App) statusLine { return a.nft.status },
			calls:  func(f *fakeServices) int { return f.getNFTCalls },
			want:   "NFT ID is required for this read.",
		},
		{
			name:   "NFT owner filter",
			run:    func(a *App) { a.filterNFTs() },
			status: func(a *App) statusLine { return a.nft.status },
			calls:  func(f *fakeServices) int { return f.listAllNFTsCalls },
			want:   "Owner is required to filter the collection.",
		},
		{
			name:   "NFT history",
			run:    func(a *App) { a.readNFTHistory() },
			status: func(a *App) statusLine { return a.nft.status },
			calls:  func(f *fakeServices) int { return f.nftHistoryCalls },
			want:   "NFT ID is required for this read.",
		},
		{
			name:   "oracle query",
			run:    func(a *App) { a.readOracleData() },
			status: func(a *App) statusLine { return a.oracle.status },
			calls:  func(f *fakeServices) int { return f.oracleDataCalls },
			want:   "Source ID is required for this read.",
		},
		{
			name:   "oracle latest",
			run:    func(a *App) { a.readOracleLatest() },
			status: func(a *App) statusLine { return a.oracle.status },
			calls:  func(f *fakeServices) int { return f.oracleLatestCalls },
			want:   "Source ID is required for this read.",
		},
		{
			name:   "session results",
			run:    func(a *App) { a.readSessionResults() },
			status: func(a *App) statusLine { return a.voting.status },
			calls:  func(f *fakeServices) int { return f.sessionResultsCalls },
			want:   "Session ID is required for this read.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeServices{}
			app := New(fake)
			if tt.prepare != nil {
				tt.prepare(app)
			}

			tt.run(app)
			got := tt.status(app)
			require.True(t, got.isErr, "incomplete %s must fail before the service call", tt.name)
			require.Contains(t, got.text, tt.want)
			require.Zero(t, tt.calls(fake), "incomplete %s must not reach the service", tt.name)
		})
	}
}

func TestCandidateRosterRequiresSelection(t *testing.T) {
	app := New(&fakeServices{candidates: []CandidateRow{{ID: "c1", Name: "Alice"}}})
	app.voting.candChoice = -1

	app.addCandidateToRoster()

	require.True(t, app.voting.status.isErr)
	require.Equal(t, "Choose a candidate first.", app.voting.status.text)
	require.Empty(t, app.voting.sessionIDs)
}

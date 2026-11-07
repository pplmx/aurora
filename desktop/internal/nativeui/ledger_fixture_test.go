package nativeui

// This file holds the ledger fixture the page tests share. It used to live only
// in the untracked zz_* screenshot scratch file, which meant a clean checkout —
// what CI has — could not compile this package at all: three tracked tests call
// captureBlocks and nothing in the tree defined it.

// captureBlocks is the ledger fixture the desktop screenshots render. The
// records are already decoded, because the view layer receives decoded rows;
// the adapter's own decoding is covered by the deskapp tests.
func captureBlocks() []BlockRow {
	const now = int64(1793548800)
	return []BlockRow{
		{
			Height: 7, Timestamp: now, Module: "lottery", Action: "Draw",
			Subject: "Alice, Carol", Summary: "3 winners from 4 participants",
			Facts: []BlockFact{
				{Label: "Winners", Value: "Alice, Carol"},
				{Label: "Participants", Value: "4"},
				{Label: "Seed", Value: "seed-abc"},
			},
		},
		{
			Height: 6, Timestamp: now - 120, Module: "token", Action: "Mint",
			Subject: "GLD", Summary: "token mint GLD owner-public-key",
			Facts: []BlockFact{
				{Label: "Token", Value: "GLD"},
				{Label: "To", Value: "owner-public-key"},
			},
		},
		{
			Height: 5, Timestamp: now - 240, Module: "voting", Action: "Ballot",
			Subject: "cand-1", Summary: "vote for cand-1",
			Facts: []BlockFact{
				{Label: "Candidate", Value: "cand-1"},
				{Label: "Voter", Value: "voter-key"},
				{Label: "Session", Value: "session-1"},
			},
		},
		{
			Height: 4, Timestamp: now - 360, Module: "oracle", Action: "Observation",
			Subject: "68421.50", Summary: "68421.50",
			Facts: []BlockFact{
				{Label: "Value", Value: "68421.50"},
				{Label: "Source", Value: "source-1"},
			},
		},
		{
			Height: 3, Timestamp: now - 480, Module: "nft", Action: "Mint",
			Subject: "Star", Summary: "nft mint Star",
			Facts: []BlockFact{
				{Label: "Name", Value: "Star"},
				{Label: "Owner", Value: "owner-public-key"},
			},
		},
	}
}

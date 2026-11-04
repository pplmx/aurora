package nativeui

import "strings"

// Jumping straight to one stored record.
//
// A record's id is what an operator copies into a ticket, a chat message or a
// colleague's window: "look at token 2026110…". Everything in the window could
// navigate to a *page*, and a deep link could name one lottery draw, but a pasted
// token, NFT, source or session id had to be walked to by hand — the page opened,
// then the collection was searched by eye. The reference apps built on MyGo treat
// this as one gesture (jump to the session by name), and the sidebar's palette is
// where a keyboard-first operator already goes to move.
//
// The resolution deliberately reuses each page's own subject path rather than
// adding a lookup: selecting a record sets fields a direct assignment would
// forget (the owner context, the readouts keyed by id, the row highlight), and a
// second implementation of that would drift from the first.

// recordKinds maps the prefix an operator types to the page that owns it. The
// prefixes are the stable page ids already used by the sidebar and deep links, so
// "token:abc" reads the same way whichever surface introduced it.
var recordKinds = map[string]page{
	"token":     pageToken,
	"nft":       pageNFT,
	"oracle":    pageOracle,
	"session":   pageVoting,
	"lottery":   pageLottery,
	"candidate": pageVoting,
	"voter":     pageVoting,
}

// parseRecordQuery splits "kind:id" into its parts. It returns ok=false for a
// query without a colon, whose kind is unknown, or whose id is empty: those are
// ordinary palette text, not a jump.
func parseRecordQuery(query string) (page, string, bool) {
	kind, id, found := strings.Cut(strings.TrimSpace(query), ":")
	if !found {
		return 0, "", false
	}
	target, ok := recordKinds[strings.ToLower(strings.TrimSpace(kind))]
	id = strings.TrimSpace(id)
	if !ok || id == "" {
		return 0, "", false
	}
	return target, id, true
}

// ShowRecord opens the page that owns a record and makes that record its
// subject, so every card below acts on it. An id the store does not know is
// still the subject: the page shows its own "not found" in the card that reads
// it, which is more useful than a palette that silently refuses to move.
func (a *App) ShowRecord(kindOrPageID, recordID string) {
	// Both spellings are accepted because both are natural: the palette has a
	// page in hand and passes its id, while a caller holding a record id says
	// what the record is ("session", not "voting"). The record kinds are checked
	// first so "session" cannot be read as a page that no longer exists.
	name := strings.ToLower(strings.TrimSpace(kindOrPageID))
	p, ok := recordKinds[name]
	if !ok {
		p, ok = pageByID(name)
	}
	if !ok {
		return
	}
	recordID = strings.TrimSpace(recordID)
	if recordID == "" {
		return
	}
	a.setPage(p)
	a.applyRecordSubject(p, recordID)
}

// applyRecordSubject hands the id to the page's own selection path. Each case is
// the same call the collection's click handler makes, so a jump and a click land
// in an identical state.
func (a *App) applyRecordSubject(p page, id string) {
	switch p {
	case pageToken:
		a.token.tokenID = id
		a.applyTokenSubject("")
		a.clearTokenHistoryForAnotherRead(id, "", 0, 0)
		a.loadTokenInfo(id)
	case pageNFT:
		a.nft.id, a.nft.transferID, a.nft.burnID, a.nft.historyID = id, id, id, id
		a.applyNFTSubject("")
		a.clearNFTDetailsForAnotherSubject(id)
		_ = a.loadNFTDetails(id)
	case pageOracle:
		a.oracle.fetchID, a.oracle.queryID, a.oracle.latestID = id, id, id
		a.oracle.tab = 2
		_ = a.loadOracleData(id, a.oracleQueryLimit())
		_ = a.loadOracleLatest(id)
	case pageVoting:
		// A voting id may be any of the three collections; the page has a field
		// per collection, and setting all of them is what the session-row click
		// does. The tab follows the id, because a tally is read on Results.
		a.voting.controlID, a.voting.voteSession = id, id
		a.voting.resultsSession = id
		a.voting.tab = 1
		a.selectSessionByID(id)
		a.clearTallyForAnotherSession(id)
		_ = a.loadSessionResults(id)
	case pageLottery:
		a.lottery.verifyID = id
		a.verifyLottery()
	}
}

package nativeui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *App) loadBlockchain() {
	a.loadBlockchainAt(a.blockchain.search, a.blockchain.blockLimit, a.blockchain.blockOffset)
}

// loadBlockchainAt loads one block query and commits its pager state only
// after the rows succeed. The status read remains a separate step so a broken
// chain warning is never replaced by an empty table.
func (a *App) loadBlockchainAt(search string, rawLimit float64, offset int) bool {
	if a.services == nil {
		return false
	}
	search = strings.TrimSpace(search)
	limit := blockchainBlockLimit(rawLimit)
	if offset < 0 {
		offset = 0
	}
	// What the rows currently on screen describe, captured before the query
	// fields move: the open payload has to be cleared when the list is about to
	// be replaced, and kept when it is only being re-read. The page size counts
	// as part of that description: changing Show keeps offset 0 and the same
	// search, yet the rows behind the index are a different set.
	previousOffset, previousSearch, previousLimit := a.blockchain.blockOffset, a.blockchain.queriedSearch, a.blockchain.queriedLimit
	a.blockchain.queriedSearch, a.blockchain.queriedLimit = search, limit
	res, err := a.services.LedgerStatus()
	a.blockchain.loadErr = err
	if err != nil {
		return false
	}
	a.blockchain.valid = res.Valid
	a.blockchain.checked = true
	a.blockchain.height = res.Height
	a.blockchain.blocks = res.Blocks
	a.blockchain.head = res.Head
	// The blocks themselves are what an auditor reads; the counters alone say
	// how much chain there is, not what it recorded. A failure here is its own
	// warning: the counters and the verdict above it are still valid.
	page, blocksErr := a.services.LedgerBlocksPage(search, limit, offset)
	a.blockchain.blocksErr = blocksErr
	if blocksErr != nil {
		return false
	}
	// The open payload belongs to a position in the list that is about to be
	// replaced. Keeping the index would point at whichever record happens to
	// sit there on the new page, so a record nobody opened would show its raw
	// payload — and the one that was open would silently close. A refresh of the
	// same page is the exception: the rows are the same and the reader expects
	// the record to stay open.
	if previousOffset != offset || previousSearch != search || previousLimit != limit {
		a.blockchain.expanded = -1
	}
	a.blockchain.rows = page.Rows
	a.blockchain.blockTotal = page.Total
	a.blockchain.blockOffset = offset
	return true
}

func blockchainBlockLimit(raw float64) int {
	limit := int(raw)
	if limit < 10 {
		return 10
	}
	if limit > 10000 {
		return 10000
	}
	return limit
}

func (a *App) blockchainView(c *ui.Context) {
	a.scrollPage(c, func() {
		guide(c, &a.guides.blockchain,
			"Every module appends its records to one hash chain; each block commits to the previous hash.",
			"The badge verifies the chain. Search matches the module, action, subject, a fact or the height across the whole chain; Show sets the page size.",
			"Open Payload to read the exact recorded bytes, then Copy payload to put them on the clipboard.",
		)
		// The page header already says what the ledger is, so the card does not
		// repeat it.
		formCard(c, "Ledger", "", func() {
			a.blockchainBody(c)
		})
		a.pendingAuditCard(c)
	})
}

func (a *App) blockchainBody(c *ui.Context) {
	if a.blockchain.loadErr != nil && !loadWarning(c, "Couldn't read the chain", a.blockchain.loadErr, a.blockchain.checked) {
		return
	}
	if !a.blockchain.checked {
		emptyNote(c, "Press Refresh to verify the chain.")
		return
	}
	// The facts and the verdict share one line: a full-width readout for two
	// short numbers left a band of empty panel above the controls that use
	// them. The tiles stay beside the verdict so the health of the chain is
	// read in the same glance as its size.
	ui.Row(c).FillWidth().Wrap().Gap(space2).AlignItems(ui.Center).Children(func() {
		statChip(c, "Blocks", strconv.Itoa(a.blockchain.blocks))
		statChip(c, "Height", strconv.Itoa(a.blockchain.height))
		if a.blockchain.head != "" {
			// The head is the chain's identity: two installations comparing a
			// record can confirm they follow the same chain without a diff.
			statChip(c, "Head", a.blockchain.head)
		}
		if a.blockchain.valid {
			verdictPill(c, true, "Hash chain verified")
		} else {
			verdictPill(c, false, "Hash chain is broken")
		}
	})
	a.blockchainBlocks(c)
}

// blockchainBlocks lists what the chain recorded, newest first. The adapter
// searches the whole chain but returns only one bounded page and its exact
// total, so Previous/Next work the same for filtered and unfiltered views.
func (a *App) blockchainBlocks(c *ui.Context) {
	limit := blockchainBlockLimit(a.blockchain.blockLimit)
	query := strings.TrimSpace(a.blockchain.search)
	queryChanged := query != a.blockchain.queriedSearch
	if queryChanged || limit != a.blockchain.queriedLimit {
		// A changed search or page size starts at the newest rows. The helper
		// records the attempt even on failure, so a database error cannot turn
		// every repaint into another read.
		a.loadBlockchainAt(query, a.blockchain.blockLimit, 0)
	}
	if a.blockchain.blocksErr != nil &&
		!loadWarning(c, "Couldn't read chain blocks", a.blockchain.blocksErr, len(a.blockchain.rows) > 0) {
		return
	}
	announceSearchResults(c, queryChanged, a.blockchain.blockTotal)
	if a.blockchain.blockTotal == 0 && query == "" {
		// Only the genesis block exists: there is nothing to search or page
		// through, and the counters above already report the block. A search
		// field over an empty list is a control that cannot work.
		emptyPanel(c, pageIcons[pageBlockchain], pageAccent(c, pageBlockchain),
			"No module records yet; the first write adds one.")
		return
	}
	// The query and the page size are one control strip: the size belongs to
	// the list the query narrows, and the search fills the row instead of
	// leaving a mostly empty band beside a field-sized input. The size is a
	// ui.Form field so its label sits beside the control it names; a stacked
	// ui.Field left the word "Show" floating above the card, a line away from
	// the number it described.
	ui.Row(c).FillWidth().Gap(space4).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Grow(1).Shrink(1).MinWidth(0).Children(func() {
			searchField(c, &a.blockchain.search, "Search blocks").
				Description(tr("Search matches the module, action, subject, a fact or the height across the whole chain.")).
				FillWidth()
		})
		ui.Row(c).Shrink(0).Children(func() {
			ui.Form(c, func() {
				ui.Field(c, tr("Show"), func() {
					ui.NumberInput(c, &a.blockchain.blockLimit, 10, 10000, 10)
				})
			})
		})
	})
	if a.blockchain.blockTotal == 0 {
		emptyNote(c, fmt.Sprintf(tr("No blocks match “%s”."), query))
		return
	}
	if len(a.blockchain.rows) == 0 {
		// The selected page no longer exists, for example after the chain was
		// reset elsewhere. Return to the first page that does.
		a.blockchain.blockOffset = 0
		a.loadBlockchainAt(query, a.blockchain.blockLimit, 0)
		return
	}
	total := a.blockchain.blockTotal
	rangeLabel := fmt.Sprintf(tr("Entries %d–%d of %d"),
		a.blockchain.blockOffset+1, a.blockchain.blockOffset+len(a.blockchain.rows), total)
	if previous, next := pagerControls(c, "Blockchain ledger", a.blockchain.blockOffset, rangeLabel,
		a.blockchain.blockOffset == 0, a.blockchain.blockOffset+len(a.blockchain.rows) >= total); previous || next {
		if previous {
			a.loadBlockchainAt(query, a.blockchain.blockLimit,
				max(0, a.blockchain.blockOffset-limit))
		} else {
			a.loadBlockchainAt(query, a.blockchain.blockLimit,
				a.blockchain.blockOffset+limit)
		}
	}
	// The ledger is a stream of records, not a grid of payload cells: each
	// entry reads as "module · action · object" with its facts and its exact
	// payload one click away. This replaced a three-column table whose
	// payload cell was the raw JSON or pipe-delimited key bytes, which made
	// the page an unreadable dump of the very thing it was meant to explain.
	//
	// A record already wraps its facts and its payload, so the same layout
	// serves the large-text preference instead of switching to a second
	// rendering of the same data.
	// The list carries the same name as its pager: a screen reader entering it
	// must say which records these are before reading the first one, and the
	// pager above already taught the reader that name.
	ui.Column(c).FillWidth().Gap(10).Role(ui.RoleList).Label(tr("Blockchain ledger")).Children(func() {
		for i, entry := range a.blockchain.rows {
			i, entry := i, entry
			blockRecord(c, entry, a.blockchain.expanded == i, func() {
				a.toggleBlockPayload(i)
			})
		}
	})
}

// toggleBlockPayload opens or closes one record's exact payload. Only one is
// open at a time: the page is a scroll of sentences, and a reader comparing
// two payloads can toggle between them faster than by scrolling past both.
func (a *App) toggleBlockPayload(index int) {
	if a.blockchain.expanded == index {
		a.blockchain.expanded = -1
		return
	}
	a.blockchain.expanded = index
}

// pageForBlockModule maps an audit payload's module label to the page that
// owns it. Unknown or synthetic blocks stay on the dashboard.
func pageForBlockModule(module string) (page, bool) {
	switch strings.ToLower(strings.TrimSpace(module)) {
	case "lottery":
		return pageLottery, true
	case "voting":
		return pageVoting, true
	case "token":
		return pageToken, true
	case "nft":
		return pageNFT, true
	case "oracle":
		return pageOracle, true
	default:
		return 0, false
	}
}

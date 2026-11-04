package nativeui

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Searching the ledger from the command palette.
//
// The palette already jumps to a page, runs a command, or opens a record by its
// exact id. The remaining case is the one an operator hits while investigating:
// they remember a word from a record — a draw's winner, a token's symbol, a
// source's name — and not its id. Searching for it meant opening Blockchain,
// typing it there, and then finding which module the result belonged to.
//
// The search is a ledger query, not a client-side filter over the loaded page: a
// match may be an old block that no page has read. It runs once per edited query
// and its result is cached, because the palette rebuilds on every frame and a
// database read per frame would be far more expensive than the feature is worth.

// paletteSearchMinRunes is how much of a query must be typed before a ledger
// search runs. One character matches most of the chain, so the first keystroke
// would issue a query whose answer is "everything".
const paletteSearchMinRunes = 2

// paletteSearchLimit bounds how many records the palette offers. The list is a
// jump target, not a report: a reader who needs to page through matches opens the
// Blockchain page, which has a pager.
const paletteSearchLimit = 5

// paletteRecordSearch is the cached ledger search behind the palette.
type paletteRecordSearch struct {
	// query is the text the rows were read for, so an edit invalidates them
	// without a second field to keep in step.
	query string
	rows  []BlockRow
	// total is every matching record in the chain, not only the rows kept. The
	// palette offers a jump target, so it draws a handful; without the total a
	// reader whose query matched forty records saw five and had no way to learn
	// the rest existed.
	total int
	err   error
	done  bool
}

// paletteRecordEntries returns the ledger matches for the current query, running
// the search only when the query changed. It returns nothing for a query that is
// too short, is a parsed id jump (which the palette already handles directly), or
// arrived while a write owns the database.
func (a *App) paletteRecordEntries(query string) []paletteEntry {
	query = strings.TrimSpace(query)
	if len([]rune(query)) < paletteSearchMinRunes || a.services == nil || a.busy != "" {
		return nil
	}
	if _, _, jumping := parseRecordQuery(query); jumping {
		return nil
	}
	if a.paletteSearch.query != query {
		a.paletteSearch = paletteRecordSearch{query: query}
		page, err := a.services.LedgerBlocksPage(query, paletteSearchLimit, 0)
		a.paletteSearch.rows, a.paletteSearch.total = page.Rows, page.Total
		a.paletteSearch.err = err
		a.paletteSearch.done = true
	}
	if a.paletteSearch.err != nil {
		return nil
	}
	entries := make([]paletteEntry, 0, len(a.paletteSearch.rows))
	for _, row := range a.paletteSearch.rows {
		target, ok := pageForBlockModule(row.Module)
		if !ok {
			continue
		}
		entries = append(entries, paletteEntry{
			title:   fmtRecordTitle(row),
			detail:  blockRecordLabel(row),
			icon:    pageIcons[target],
			matched: true,
			action:  func(*ui.Context) { a.navigateToRecord(target, row) },
		})
	}
	// Say when the list is cut short, and give the cut its own way out. A
	// truncated list that stays quiet is worse than a short one: the reader
	// concludes their record is not in the chain rather than that the palette
	// shows only part of it.
	if remaining := a.paletteSearch.total - len(entries); remaining > 0 {
		query := query
		entries = append(entries, paletteEntry{
			title:   fmt.Sprintf(tr("Show all %d matching records"), a.paletteSearch.total),
			detail:  fmt.Sprintf(tr("Opens Blockchain with “%s” in the search field"), query),
			icon:    pageIcons[pageBlockchain],
			matched: true,
			action:  func(*ui.Context) { a.showLedgerSearch(query) },
		})
	}
	return entries
}

// showLedgerSearch opens the ledger page with the palette's query already in its
// search field, so the pager that the palette does not have is one click away
// from the truncated list.
func (a *App) showLedgerSearch(query string) {
	a.blockchain.search = query
	a.blockchain.blockOffset = 0
	a.setPage(pageBlockchain)
	if a.invalidate != nil {
		a.invalidate()
	}
}

// fmtRecordTitle is the row's headline: the module's own name and the action it
// took, which is what distinguishes two records from the same module.
func fmtRecordTitle(row BlockRow) string {
	title := recordModuleName(row.Module)
	if action := recordActionName(row.Module, row.Action); action != "" {
		title += " · " + action
	}
	return title
}

// navigateToRecord opens the module that wrote a record. It does not try to make
// the record the subject: a block carries the payload the modules wrote, not
// necessarily a token or NFT id the page can select, and guessing one from the
// decoded facts would sometimes select the wrong record.
func (a *App) navigateToRecord(target page, row BlockRow) {
	a.setPage(target)
	if a.invalidate != nil {
		a.invalidate()
	}
}

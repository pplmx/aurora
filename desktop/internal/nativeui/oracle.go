package nativeui

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// maxOracleIntervalSeconds is the longest polling interval the form accepts,
// matching the domain's 30-day ceiling: a larger value overflows the
// scheduler's due-time arithmetic into "always due".
const maxOracleIntervalSeconds = 30 * 24 * 60 * 60

// oracleState is the Oracle page's state.
type oracleState struct {
	// Add-source form.
	addName     string
	addURL      string
	addType     string
	addMethod   string
	addPath     string
	addHeaders  string
	addInterval float64
	template    string
	// addAdvanced holds the transport fields most feeds do not set on the
	// first try: type, method, JSON path, headers and polling interval.
	addAdvanced bool

	// Per-source actions.
	fetchID  string
	queryID  string
	queryLim float64
	latestID string

	sources   []SourceRow
	templates []TemplateRow
	health    []HealthRow
	// offset pages the source collection. hasMore comes from one lookahead
	// row so a full final page does not leave a dead Next control.
	offset  int
	hasMore bool
	// healthErr is the last failure to read the health counters; the rows from
	// the previous successful read stay on screen with a stale note.
	healthErr error
	data      []DataRow
	latest    *DataRow
	// The Query and Latest cards are caches keyed by the source (and, for
	// Query, the limit). A selection or fetch changes those inputs; without
	// the key, a failed read left the previous source's value under the new
	// id. The error keeps a failed same-input refresh visible.
	dataErr            error
	dataLoadedSource   string
	dataLoadedLimit    int
	latestErr          error
	latestLoadedSource string

	loadErr error
	// templatesErr is the last failure to read the preset catalog; old presets
	// stay usable with a warning.
	templatesErr error

	// selected is the source row chosen, -1 for none.
	selected int
	// search narrows the source list by name, id, type or URL; appliedSearch
	// is the query the current highlight belongs to.
	search        string
	appliedSearch string

	// askDelete drives the source-deletion confirmation: deleting a source is
	// irreversible for its scheduling entry, so it gets the same gate as burn.
	askDelete bool
	// deleteName is resolved when the dialog opens. The action field accepts
	// ids from the CLI or a copied page, so the target may be outside the
	// currently loaded source page.
	deleteName string

	// tab is the right-hand section: 0 sources (list/actions), 1 feed health,
	// 2 readings (query/latest). Stacking all four made the page a long column
	// where the read cards were usually below the fold.
	tab int

	status statusLine

	// deleteTrigger receives focus when the destructive confirmation closes,
	// keeping keyboard navigation on the source action row.
	deleteTrigger ui.Handle
}

const oraclePageSize = 100

func (a *App) loadOracle() {
	if a.services == nil {
		return
	}
	selectedID := a.selectedSourceID()
	a.loadOracleSourcePage(a.oracle.offset)
	a.restoreSourceChoice(selectedID)
	a.loadOracleAuxiliary()
}

// loadOracleAuxiliary refreshes the two lists whose failures are independent
// of the source page. Keeping them separate lets a successful source add retry
// page zero without losing the current page when that read fails.
func (a *App) loadOracleAuxiliary() {
	// Each list has its own failure field and its own last successful rows.
	// A source-list failure must not freeze the health counters or the preset
	// picker as well.
	if tpl, err := a.services.OracleTemplates(); err != nil {
		a.oracle.templatesErr = err
	} else {
		a.oracle.templates = tpl
		a.oracle.templatesErr = nil
	}
	if h, err := a.services.OracleHealth(); err != nil {
		a.oracle.healthErr = err
	} else {
		a.oracle.health = h
		a.oracle.healthErr = nil
	}
}

// loadOracleSourcePage commits offset only after the read succeeds, so a
// failed pager move leaves the rows and highlight on their actual page.
func (a *App) loadOracleSourcePage(offset int) bool {
	if a.services == nil {
		return false
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := a.services.OracleSources(oraclePageSize+1, offset)
	a.oracle.loadErr = err
	if err != nil {
		return false
	}
	if len(rows) == 0 && offset > 0 {
		return a.loadOracleSourcePage(0)
	}
	a.oracle.offset = offset
	a.oracle.hasMore = len(rows) > oraclePageSize
	if a.oracle.hasMore {
		rows = rows[:oraclePageSize]
	}
	a.oracle.sources = rows
	return true
}

func (a *App) selectedSourceID() string {
	rows := a.filteredSources()
	if a.oracle.selected < 0 || a.oracle.selected >= len(rows) {
		return ""
	}
	return rows[a.oracle.selected].ID
}

func (a *App) restoreSourceChoice(id string) {
	a.oracle.selected = -1
	if id == "" {
		return
	}
	for i, source := range a.filteredSources() {
		if source.ID == id {
			a.oracle.selected = i
			return
		}
	}
}

func (a *App) oracleView(c *ui.Context) {
	a.scrollPage(c, func() {
		if a.oracleNeedsFirstRecord() {
			twoColumns(c, func() {
				a.oracleAdd(c)
			}, func() {
				firstRunPath(c, pageOracle, "No sources yet",
					"A source begins with a URL; Aurora can poll it, record readings and report health.",
					"Fill in the add-source form.",
					"Choose the new source to fetch it now.",
					"Review scheduler health and stored values.",
				)
			})
			a.renderDeleteDialog(c)
			return
		}
		// The first-record panel already explains the feed's first three moves.
		guide(c, &a.guides.oracle,
			"Add a source with a name and URL. Advanced holds the method, JSON path, headers and polling interval; a preset fills them for review.",
			"Select a source in Sources to fetch it now, enable/disable it, or delete it.",
			"Health shows the scheduler's attempts and last error; Readings queries stored values or the latest one.",
		)
		// The add form is the tallest card on the page, so it gets a column of
		// its own; the right column shows one of sources, health or readings at
		// a time instead of stacking all four sections.
		left := func() {
			a.oracleAdd(c)
			a.oracleSourceActions(c)
		}
		stage := func() {
			tabs(c, &a.oracle.tab, "Oracle sections", tr("Sources"), tr("Health"), tr("Readings"))
			switch a.oracle.tab {
			case 0:
				a.oracleSources(c)
			case 1:
				a.oracleHealth(c)
			default:
				a.loadOpenReadings()
				a.oracleRead(c)
			}
		}
		// Unlike the other modules, this page's first column is setup rather
		// than the list the stage acts on. Stacked (a narrow window or large
		// text), drawing the tall add form first would push the stage selector
		// two screens down, so the stage leads and the setup follows.
		if sideBySide(c) {
			twoColumns(c, left, stage)
		} else {
			twoColumns(c, stage, left)
		}
		// The action card lives in the left column and is visible on every
		// stage, so its confirmation must be mounted once for the page rather
		// than from the Sources tab body.
		a.renderDeleteDialog(c)
	})
	if !a.oracle.askDelete {
		a.liveRefresh(c, a.refreshOracleLive)
	}
}

// oracleNeedsFirstRecord applies the same first-write hierarchy to feeds.
// Existing sources, an open stage, a filter or a source id all mean the
// operator is already working with a collection, so the full page stays.
func (a *App) oracleNeedsFirstRecord() bool {
	entered := a.oracle.tab != 0 ||
		strings.TrimSpace(a.oracle.search) != "" || a.oracle.appliedSearch != "" ||
		a.oracle.selected >= 0 ||
		strings.TrimSpace(a.oracle.fetchID) != "" ||
		strings.TrimSpace(a.oracle.queryID) != "" ||
		strings.TrimSpace(a.oracle.latestID) != "" ||
		a.oracle.data != nil || a.oracle.latest != nil ||
		a.oracle.dataLoadedSource != "" || a.oracle.latestLoadedSource != ""
	return firstRecordOnly(a.oracle.loadErr, len(a.oracle.sources), entered)
}

// refreshOracleLive re-reads the scheduler's health and the lists the page
// shows. It does not touch the add form or a read card the user has not opened.
func (a *App) refreshOracleLive() {
	a.loadOracle()
	a.refreshOracleReads()
}

// sourceInterval renders a source's polling cadence. A non-positive interval
// means the scheduler never polls it, which is different from a very long one.
func sourceInterval(seconds int) string {
	if seconds <= 0 {
		return tr("not polled")
	}
	return strconv.Itoa(seconds) + "s"
}

// oracleQueryLimit is the Query card's limit: 20 unless the field names a
// positive number.
func (a *App) oracleQueryLimit() int {
	limit := int(a.oracle.queryLim)
	if limit < 1 {
		return 20
	}
	if limit > maxListLimit {
		return maxListLimit
	}
	return limit
}

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

	// Per-source actions.
	fetchID  string
	queryID  string
	queryLim float64
	latestID string

	sources   []SourceRow
	templates []TemplateRow
	health    []HealthRow
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

	status statusLine

	sourceTable ui.ListState
	healthTable ui.ListState
	dataTable   ui.ListState
}

func (a *App) loadOracle() {
	if a.services == nil {
		return
	}
	// Each list has its own failure field and its own last successful rows.
	// Reading sources no longer returns early: a source-list failure must not
	// freeze the health counters or the preset picker as well.
	if sources, err := a.services.OracleSources(); err != nil {
		a.oracle.loadErr = err
	} else {
		a.oracle.sources = sources
		a.oracle.loadErr = nil
	}
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

func (a *App) oracleView(c *ui.Context) {
	ui.Scroll(c).Fill().Children(func() {
		// The add form is the tallest card on the page, so it gets a column of
		// its own; the short read cards fill the other one.
		twoColumns(c, func() {
			a.oracleAdd(c)
		}, func() {
			a.oracleSources(c)
			a.oracleHealth(c)
			a.oracleRead(c)
		})
	})
}

func (a *App) oracleSources(c *ui.Context) {
	formCard(c, "Data Sources", "Every registered feed, its transport, and whether it is currently polled.", func() {
		a.oracleSourcesBody(c)
	})
}

func (a *App) oracleSourcesBody(c *ui.Context) {
	if a.oracle.loadErr != nil && !loadWarning(c, "Failed to load sources", a.oracle.loadErr, len(a.oracle.sources) > 0) {
		return
	}
	if len(a.oracle.sources) == 0 {
		emptyNote(c, "No sources yet. Add one with Add Source.")
		return
	}
	srcs := make([]SourceRow, 0, len(a.oracle.sources))
	ui.SearchField(c, &a.oracle.search).Label("Search sources")
	if a.oracle.search != a.oracle.appliedSearch {
		a.oracle.appliedSearch = a.oracle.search
		a.oracle.selected = -1
	}
	for _, source := range a.oracle.sources {
		if containsFold(a.oracle.search, source.ID, source.Name, source.Type, source.URL) {
			srcs = append(srcs, source)
		}
	}
	if len(srcs) == 0 {
		emptyNote(c, "No sources match “"+strings.TrimSpace(a.oracle.search)+"”.")
		return
	}
	// The id is a timestamp plus a hash, too long for a column that also holds
	// a name and a status, so the cell shows a prefix. Choosing the row loads
	// the full id into the actions and the read cards below, which is how a
	// source reaches them without retyping an id nobody has in full.
	a.oracle.sourceTable.Selected = &a.oracle.selected
	table := ui.Table(c, &a.oracle.sourceTable, []ui.TableColumn{
		{Title: "Name", MinWidth: 110},
		{Title: "Type", Width: 100},
		{Title: "ID", Width: 150},
		{Title: "Status", Width: 100},
		{Title: "Headers", Width: 100},
	}, len(srcs), func(row, col int) {
		s := srcs[row]
		switch col {
		case 0:
			ui.Text(c, s.Name).SingleLine()
		case 1:
			ui.Text(c, s.Type)
		case 2:
			ui.Text(c, shortID(s.ID)).SingleLine()
		case 3:
			if s.Enabled {
				ui.Text(c, "enabled")
			} else {
				ui.Text(c, "disabled")
			}
		case 4:
			if s.HasHeaders {
				ui.Text(c, "configured")
			} else {
				ui.Text(c, "—")
			}
		}
	})
	if table.Changed() && a.oracle.selected >= 0 && a.oracle.selected < len(srcs) {
		id := srcs[a.oracle.selected].ID
		a.oracle.fetchID, a.oracle.queryID, a.oracle.latestID = id, id, id
		// The Query and Latest cards follow the selection when they are open,
		// instead of showing the previously chosen source.
		a.refreshOracleReads()
	}
	if a.oracle.selected >= 0 && a.oracle.selected < len(srcs) {
		// The row is too narrow for the endpoint and the polling cadence, the
		// two facts that explain when and where the source fetches.
		chosen := srcs[a.oracle.selected]
		copyRow(c, "URL", chosen.URL, nil)
		readout(c, []readoutRow{{"Every", sourceInterval(chosen.Interval)}})
	}

	// Per-source actions. A table row cannot host a button per cell without
	// making the whole grid interactive, so the id is entered once — or loaded
	// by choosing a row above — and the actions apply to it.
	field(c, "Source ID for the actions below", &a.oracle.fetchID)
	ui.Row(c).Gap(8).Children(func() {
		if secondaryButton(c, "Enable").Clicked() {
			a.oracle.status = wrapErr(a.services.SetOracleSourceEnabled(strings.TrimSpace(a.oracle.fetchID), true))
			a.loadOracle()
		}
		if secondaryButton(c, "Disable").Clicked() {
			a.oracle.status = wrapErr(a.services.SetOracleSourceEnabled(strings.TrimSpace(a.oracle.fetchID), false))
			a.loadOracle()
		}
		if secondaryButton(c, "Fetch now").Clicked() {
			sourceID := strings.TrimSpace(a.oracle.fetchID)
			err := a.services.FetchOracleSource(sourceID)
			a.oracle.status = wrapErr(err)
			a.loadOracle()
			if err == nil {
				// A reading just stored must appear in the read cards already
				// showing that source; leaving the old value beside "Done"
				// reads as a fetch that stored nothing.
				if strings.TrimSpace(a.oracle.queryID) == sourceID &&
					(len(a.oracle.data) > 0 || a.oracle.dataErr != nil) {
					limit := a.oracleQueryLimit()
					a.clearOracleDataForAnotherRead(sourceID, limit)
					_ = a.loadOracleData(sourceID, limit)
				}
				if strings.TrimSpace(a.oracle.latestID) == sourceID &&
					(a.oracle.latest != nil || a.oracle.latestErr != nil) {
					_ = a.loadOracleLatest(sourceID)
				}
			}
		}
		if dangerButton(c, "Delete").Clicked() {
			a.oracle.askDelete = true
		}
	})
	a.renderDeleteDialog(c)
}

func (a *App) renderDeleteDialog(c *ui.Context) {
	if !a.oracle.askDelete {
		return
	}
	// Name the source being deleted, from the loaded list when it is there:
	// the actions share one id field, and "Delete this source?" gave the
	// operator nothing to check it against.
	detail := "The source and its scheduling entry are removed. Readings already stored stay."
	if id := strings.TrimSpace(a.oracle.fetchID); id != "" {
		target := shortID(id)
		for _, source := range a.oracle.sources {
			if source.ID == id {
				target = source.Name + " (" + shortID(id) + ")"
				break
			}
		}
		detail = "Delete " + target + "? Its scheduling entry is removed; readings already stored stay."
	}
	switch ui.AlertDialog(c, &a.oracle.askDelete, "Delete this source?", detail, "Cancel", "Delete") {
	case 1:
		a.deleteOracleSource()
	}
}

// deleteOracleSource removes the source named by the action field. The cards
// showing its readings are cleared when they belong to it: their rows are gone
// from the source's point of view, and leaving them beside the deletion result
// would suggest the readings still describe a live feed.
func (a *App) deleteOracleSource() {
	id := strings.TrimSpace(a.oracle.fetchID)
	err := a.services.DeleteOracleSource(id)
	a.oracle.status = wrapErr(err)
	if err == nil {
		if strings.TrimSpace(a.oracle.queryID) == id {
			a.oracle.data = nil
			a.oracle.dataErr = nil
			a.oracle.dataLoadedSource = ""
		}
		if strings.TrimSpace(a.oracle.latestID) == id {
			a.oracle.latest = nil
			a.oracle.latestErr = nil
			a.oracle.latestLoadedSource = ""
		}
	}
	a.loadOracle()
}

func (a *App) oracleAdd(c *ui.Context) {
	formCard(c, "Add Source", "Point Aurora at an HTTP endpoint; the interval controls how often it is polled.", func() {
		a.oracleAddBody(c)
	})
}

func (a *App) oracleAddBody(c *ui.Context) {
	if a.oracle.templatesErr != nil {
		// The form stays usable without presets; the warning explains why the
		// picker is stale or absent instead of silently dropping it.
		loadWarning(c, "Failed to load presets", a.oracle.templatesErr, len(a.oracle.templates) > 0)
	}
	if len(a.oracle.templates) > 0 {
		// The choice lives in the page's state: a local variable reset it to
		// empty on the next frame, so the box never showed what was picked.
		// It is labelled, because a bare box above "Name" read as a Name
		// field with no caption.
		names := make([]string, 0, len(a.oracle.templates)+1)
		names = append(names, "— choose a preset —")
		for _, t := range a.oracle.templates {
			names = append(names, t.ID+" — "+t.Name)
		}
		ui.Field(c, "Preset (optional)", func() {
			if ui.Combobox(c, &a.oracle.template, names).Changed() &&
				a.oracle.template != "" && a.oracle.template != names[0] {
				a.applyTemplate(strings.SplitN(a.oracle.template, " — ", 2)[0])
			}
		})
	}
	field(c, "Name", &a.oracle.addName)
	field(c, "URL", &a.oracle.addURL)
	field(c, "Type (optional)", &a.oracle.addType)
	field(c, "Method (optional, defaults to GET)", &a.oracle.addMethod)
	field(c, "Path (optional JSON field)", &a.oracle.addPath)
	textAreaHint(c, "Request Headers (optional JSON object)",
		"Values are stored as request headers and shown as “configured”.", &a.oracle.addHeaders)
	ui.Field(c, "Interval (seconds)", func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.NumberInput(c, &a.oracle.addInterval, 0, maxOracleIntervalSeconds, 1)
			hint(c, "0 uses the default of 60; at most 30 days.")
		})
	})
	ui.Row(c).Children(func() {
		if primaryButton(c, "Add Source").Clicked() {
			a.addOracleSource()
		}
	})
}

// applyTemplate fills the Add Source form from a preset: every field the
// preset defines, so the operator can review and add it. Filling only the name
// left the form without a URL, and Add Source then refused a preset that
// `oracle source template` adds without complaint.
func (a *App) applyTemplate(id string) {
	for _, t := range a.oracle.templates {
		if t.ID != id {
			continue
		}
		a.oracle.addName = t.Name
		a.oracle.addType = t.ID
		a.oracle.addURL = t.URL
		a.oracle.addMethod = t.Method
		a.oracle.addPath = t.Path
		a.oracle.addInterval = float64(t.Interval)
		return
	}
}

func (a *App) addOracleSource() {
	interval := int(a.oracle.addInterval)
	src, err := a.services.AddOracleSource(SourceInput{
		Name:     strings.TrimSpace(a.oracle.addName),
		URL:      strings.TrimSpace(a.oracle.addURL),
		Type:     strings.TrimSpace(a.oracle.addType),
		Method:   strings.TrimSpace(a.oracle.addMethod),
		Path:     strings.TrimSpace(a.oracle.addPath),
		Headers:  strings.TrimSpace(a.oracle.addHeaders),
		Interval: interval,
	})
	if err != nil {
		a.oracle.status = errStatus(err)
		return
	}
	a.oracle.status = okStatus("Added %s (%s)", src.Name, src.ID)
	a.oracle.addName, a.oracle.addURL, a.oracle.addType = "", "", ""
	a.oracle.addMethod, a.oracle.addPath, a.oracle.addHeaders = "", "", ""
	a.oracle.addInterval = 0
	// The preset choice is part of the form, not a permanent mode: leaving it
	// selected would show a preset whose fields were just cleared, and picking
	// it again would not report a change.
	a.oracle.template = ""
	a.loadOracle()
}

func (a *App) oracleHealth(c *ui.Context) {
	formCard(c, "Feed Health", "Background poll attempts, successes and failures per source, with the last error kept verbatim.", func() {
		a.oracleHealthBody(c)
	})
}

func (a *App) oracleHealthBody(c *ui.Context) {
	if a.oracle.healthErr != nil &&
		!loadWarning(c, "Failed to load feed health", a.oracle.healthErr, len(a.oracle.health) > 0) {
		return
	}
	if len(a.oracle.health) == 0 {
		emptyNote(c, "(no feed activity yet)")
		return
	}
	health := a.oracle.health
	// The scheduler reports ids. The page already loaded the source list, so a
	// name comes from there instead of a second read; an id the list no longer
	// holds (deleted between reads) still identifies its row.
	names := make(map[string]string, len(a.oracle.sources))
	for _, source := range a.oracle.sources {
		names[source.ID] = source.Name
	}
	ui.Table(c, &a.oracle.healthTable, []ui.TableColumn{
		{Title: "Source"},
		{Title: "Attempts", Width: 80, Align: ui.End},
		{Title: "Successes", Width: 80, Align: ui.End},
		{Title: "Failures", Width: 80, Align: ui.End},
		{Title: "Last error"},
	}, len(health), func(row, col int) {
		h := health[row]
		switch col {
		case 0:
			name := h.Name
			if name == "" {
				name = names[h.SourceID]
			}
			if name == "" {
				name = h.SourceID
			}
			ui.Text(c, name)
		case 1:
			ui.Text(c, strconv.Itoa(h.Attempts))
		case 2:
			ui.Text(c, strconv.Itoa(h.Successes))
		case 3:
			ui.Text(c, strconv.Itoa(h.Failures))
		case 4:
			if h.LastError == "" {
				ui.Text(c, "—")
			} else {
				ui.Text(c, h.LastError)
			}
		}
	})
}

func (a *App) oracleRead(c *ui.Context) {
	formCard(c, "Query Data", "Read the most recent stored values for one source.", func() {
		a.oracleReadBody(c)
	})
	a.oracleLatest(c)
}

func (a *App) oracleReadBody(c *ui.Context) {
	field(c, "Source ID", &a.oracle.queryID)
	ui.Field(c, "Limit", func() {
		ui.NumberInput(c, &a.oracle.queryLim, 1, maxListLimit, 1)
	})
	ui.Row(c).Children(func() {
		if secondaryButton(c, "Query").Clicked() {
			sourceID := strings.TrimSpace(a.oracle.queryID)
			limit := a.oracleQueryLimit()
			a.clearOracleDataForAnotherRead(sourceID, limit)
			if err := a.loadOracleData(sourceID, limit); err != nil {
				a.oracle.status = errStatus(err)
			} else {
				a.oracle.status = statusLine{}
			}
		}
	})
	if a.oracle.dataErr != nil &&
		!loadWarning(c, "Couldn't load readings", a.oracle.dataErr, len(a.oracle.data) > 0) {
		return
	}
	if len(a.oracle.data) > 0 {
		data := a.oracle.data
		ui.Table(c, &a.oracle.dataTable, []ui.TableColumn{
			{Title: "Value"},
			{Title: "Block", Width: 80, Align: ui.End},
		}, len(data), func(row, col int) {
			d := data[row]
			if col == 0 {
				ui.Text(c, d.Value)
			} else {
				ui.Text(c, strconv.FormatInt(d.BlockHeight, 10))
			}
		})
	}
}

// refreshOracleReads re-reads the Query and Latest cards that are already open
// (they hold rows or a failure). The key check invalidates a card whose source
// or limit changed, so a failed read cannot leave the previous answer under
// the new input.
func (a *App) refreshOracleReads() {
	if len(a.oracle.data) > 0 || a.oracle.dataErr != nil {
		sourceID := strings.TrimSpace(a.oracle.queryID)
		limit := a.oracleQueryLimit()
		a.clearOracleDataForAnotherRead(sourceID, limit)
		_ = a.loadOracleData(sourceID, limit)
	}
	if a.oracle.latest != nil || a.oracle.latestErr != nil {
		sourceID := strings.TrimSpace(a.oracle.latestID)
		a.clearOracleLatestForAnotherSource(sourceID)
		_ = a.loadOracleLatest(sourceID)
	}
}

// sourceInterval renders a source's polling cadence. A non-positive interval
// means the scheduler never polls it, which is different from a very long one.
func sourceInterval(seconds int) string {
	if seconds <= 0 {
		return "not polled"
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

// loadOracleData reads a source's recent values into the Query card. A failure
// keeps the rows only when the same source and limit produced them; the caller
// invalidates them when either input changes.
func (a *App) loadOracleData(sourceID string, limit int) error {
	rows, err := a.services.OracleData(sourceID, limit)
	if err != nil {
		a.oracle.dataErr = err
		a.oracle.dataLoadedSource, a.oracle.dataLoadedLimit = sourceID, limit
		return err
	}
	a.oracle.data = rows
	a.oracle.dataLoadedSource, a.oracle.dataLoadedLimit = sourceID, limit
	a.oracle.dataErr = nil
	return nil
}

func (a *App) clearOracleDataForAnotherRead(sourceID string, limit int) {
	if a.oracle.dataLoadedSource == sourceID && a.oracle.dataLoadedLimit == limit {
		return
	}
	a.oracle.data = nil
	a.oracle.dataErr = nil
	a.oracle.dataLoadedSource = ""
}

// loadOracleLatest reads a source's newest value into the Latest card. The
// source is the key: a failure keeps the value only for the same source.
func (a *App) loadOracleLatest(sourceID string) error {
	data, err := a.services.OracleLatest(sourceID)
	if err != nil {
		a.oracle.latestErr = err
		a.oracle.latestLoadedSource = sourceID
		return err
	}
	a.oracle.latest = &data
	a.oracle.latestLoadedSource = sourceID
	a.oracle.latestErr = nil
	return nil
}

func (a *App) clearOracleLatestForAnotherSource(sourceID string) {
	if a.oracle.latestLoadedSource == sourceID {
		return
	}
	a.oracle.latest = nil
	a.oracle.latestErr = nil
	a.oracle.latestLoadedSource = ""
}

// oracleLatest draws the single-value read. It is its own section rather than
// the tail of Query Data: a query that returns nothing used to return early
// and hide this form entirely, so the one control that answers "what is the
// current value?" was unreachable on a database with no stored readings.
func (a *App) oracleLatest(c *ui.Context) {
	formCard(c, "Latest Data", "The single most recent value recorded for a source.", func() {
		a.oracleLatestBody(c)
	})
}

func (a *App) oracleLatestBody(c *ui.Context) {
	field(c, "Source ID", &a.oracle.latestID)
	ui.Row(c).Children(func() {
		if secondaryButton(c, "Get Latest").Clicked() {
			sourceID := strings.TrimSpace(a.oracle.latestID)
			a.clearOracleLatestForAnotherSource(sourceID)
			if err := a.loadOracleLatest(sourceID); err != nil {
				a.oracle.status = errStatus(err)
			} else {
				a.oracle.status = statusLine{}
			}
		}
	})
	if a.oracle.latestErr != nil &&
		!loadWarning(c, "Couldn't load the latest value", a.oracle.latestErr, a.oracle.latest != nil) {
		return
	}
	if l := a.oracle.latest; l != nil {
		readout(c, []readoutRow{{"Value", l.Value}, {"Block", strconv.FormatInt(l.BlockHeight, 10)}})
	}
}

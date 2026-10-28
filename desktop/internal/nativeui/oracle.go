package nativeui

import (
	"fmt"
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

	sourceTable ui.ListState
	healthTable ui.ListState
	dataTable   ui.ListState
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
		guide(c, &a.guides.oracle,
			"Add a source with a name and URL. Advanced holds the method, JSON path, headers and polling interval; a preset fills them for review.",
			"Select a source in Sources to fetch it now, enable/disable it, or delete it.",
			"Health shows the scheduler's attempts and last error; Readings queries stored values or the latest one.",
		)
		// The add form is the tallest card on the page, so it gets a column of
		// its own; the right column shows one of sources, health or readings at
		// a time instead of stacking all four sections.
		twoColumns(c, func() {
			a.oracleAdd(c)
		}, func() {
			tabs(c, &a.oracle.tab, tr("Sources"), tr("Health"), tr("Readings"))
			switch a.oracle.tab {
			case 0:
				a.oracleSources(c)
			case 1:
				a.oracleHealth(c)
			default:
				a.loadOpenReadings()
				a.oracleRead(c)
			}
		})
	})
	if !a.oracle.askDelete {
		a.liveRefresh(c, a.refreshOracleLive)
	}
}

// refreshOracleLive re-reads the scheduler's health and the lists the page
// shows. It does not touch the add form or a read card the user has not opened.
func (a *App) refreshOracleLive() {
	a.loadOracle()
	a.refreshOracleReads()
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
		emptyState(c, pageOracle, "No sources yet", "Add an HTTP feed on the left to start reading values.")
		return
	}
	ui.SearchField(c, &a.oracle.search).Label(tr("Search sources"))
	if a.oracle.search != a.oracle.appliedSearch {
		a.oracle.appliedSearch = a.oracle.search
		a.oracle.selected = -1
	}
	if previous, next := listPager(c, a.oracle.offset, len(a.oracle.sources), a.oracle.hasMore); previous || next {
		target := a.oracle.offset
		if previous {
			target = max(0, target-oraclePageSize)
		} else {
			target += oraclePageSize
		}
		if a.loadOracleSourcePage(target) {
			a.oracle.selected = -1
		}
		return
	}
	if a.oracle.search != "" {
		hint(c, "Search narrows the current page; use Previous/Next for older sources.")
	}
	srcs := a.filteredSources()
	if len(srcs) == 0 {
		emptyNote(c, fmt.Sprintf(tr("No sources match “%s”."), strings.TrimSpace(a.oracle.search)))
		return
	}
	// The id is a timestamp plus a hash, too long for a column that also holds
	// a name and a status, so the cell shows a prefix. Choosing the row loads
	// the full id into the actions and the read cards below, which is how a
	// source reaches them without retyping an id nobody has in full.
	if stackedRecords(c) {
		a.oracleSourceRecords(c, srcs)
	} else {
		a.oracle.sourceTable.Selected = &a.oracle.selected
		table := ui.Table(c, &a.oracle.sourceTable, []ui.TableColumn{
			{Title: tr("Name"), MinWidth: 110},
			{Title: tr("Type"), Width: 100},
			{Title: tr("ID"), Width: 150},
			{Title: tr("Status"), Width: 100},
			{Title: tr("Headers"), Width: 100},
		}, len(srcs), func(row, col int) {
			s := srcs[row]
			switch col {
			case 0:
				tableText(c, s.Name)
			case 1:
				tableText(c, s.Type)
			case 2:
				tableTextFull(c, shortID(s.ID), s.ID)
			case 3:
				if s.Enabled {
					tableText(c, tr("enabled"))
				} else {
					tableText(c, tr("disabled"))
				}
			case 4:
				if s.HasHeaders {
					tableText(c, tr("configured"))
				} else {
					tableText(c, tr("—"))
				}
			}
		})
		if table.Changed() {
			a.chooseOracleSource(srcs, a.oracle.selected)
		}
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
	copyRow(c, "Source ID", strings.TrimSpace(a.oracle.fetchID), nil)
	ui.Row(c).Wrap().Gap(8).Children(func() {
		if secondaryButton(c, "Enable").Clicked() {
			a.setOracleEnabled(true)
		}
		if secondaryButton(c, "Disable").Clicked() {
			a.setOracleEnabled(false)
		}
		if secondaryButton(c, "Fetch now").Clicked() {
			a.fetchOracleSource()
		}
		if dangerButton(c, "Delete").Clicked() {
			a.oracle.deleteName = a.resolveOracleSourceName(strings.TrimSpace(a.oracle.fetchID))
			a.oracle.askDelete = true
		}
	})
	a.renderDeleteDialog(c)
}

// oracleSourceRecords preserves the complete source id and all four status
// columns when enlarged text no longer fits the five-column table.
func (a *App) oracleSourceRecords(c *ui.Context, sources []SourceRow) {
	ui.Column(c).FillWidth().Gap(10).Children(func() {
		for i, source := range sources {
			i, source := i, source
			displayName := source.Name
			if strings.TrimSpace(displayName) == "" {
				displayName = shortID(source.ID)
			}
			status, tone := tr("disabled"), inkMuted
			if source.Enabled {
				status, tone = tr("enabled"), auroraGreen
			}
			headers := tr("—")
			if source.HasHeaders {
				headers = tr("configured")
			}
			selected := i == a.oracle.selected
			record := selectableRecordPanel(c, selected,
				recordChoiceLabel(displayName, selected), func() {
					ui.Text(c, displayName).FontSize(textSize(c, 13)).Bold().
						TextColor(ui.Hex(ink))
					ui.Row(c).Wrap().Gap(18).Children(func() {
						recordMetric(c, "Type", valueOr(source.Type), ink)
						recordMetric(c, "Status", status, tone)
						recordMetric(c, "Headers", headers, ink)
					})
					recordField(c, "ID", source.ID)
				})
			if record.Clicked() {
				a.chooseOracleSource(sources, i)
			}
		}
	})
}

// chooseOracleSource is shared by the table and records. It loads the exact id
// into every action and refreshes the read cards already open for a source.
func (a *App) chooseOracleSource(sources []SourceRow, index int) {
	if index < 0 || index >= len(sources) {
		return
	}
	a.oracle.selected = index
	id := sources[index].ID
	a.oracle.fetchID, a.oracle.queryID, a.oracle.latestID = id, id, id
	a.refreshOracleReads()
}

func (a *App) filteredSources() []SourceRow {
	rows := make([]SourceRow, 0, len(a.oracle.sources))
	for _, source := range a.oracle.sources {
		if containsFold(a.oracle.search, source.ID, source.Name, source.Type, source.URL) {
			rows = append(rows, source)
		}
	}
	return rows
}

func (a *App) renderDeleteDialog(c *ui.Context) {
	if !a.oracle.askDelete {
		return
	}
	// Name the source being deleted. The action field can name a source outside
	// the paged list, so the label is resolved when the dialog opens rather
	// than only by scanning the visible rows.
	detail := tr("The source and its scheduling entry are removed. Readings already stored stay.")
	if id := strings.TrimSpace(a.oracle.fetchID); id != "" {
		target := shortID(id)
		if name := strings.TrimSpace(a.oracle.deleteName); name != "" {
			target = conciseText(name, 32) + " (" + shortID(id) + ")"
		}
		detail = fmt.Sprintf(tr("Delete %s? Its scheduling entry is removed; readings already stored stay."), target)
	}
	switch ui.AlertDialog(c, &a.oracle.askDelete, tr("Delete this source?"), detail, tr("Cancel"), tr("Delete")) {
	case 1:
		a.deleteOracleSource()
	}
}

// resolveOracleSourceName finds the operator-facing name for the confirmation.
// The source collection is paged, while the action field also accepts ids from
// the CLI or a previous copy, so a page-only lookup reduced older sources to a
// short opaque id. A failed lookup leaves that id fallback intact rather than
// blocking the destructive action.
func (a *App) resolveOracleSourceName(id string) string {
	if id == "" {
		return ""
	}
	for _, source := range a.oracle.sources {
		if source.ID == id {
			return source.Name
		}
	}
	if a.services == nil {
		return ""
	}
	source, err := a.services.OracleSource(id)
	if err != nil {
		return ""
	}
	return source.Name
}

// deleteOracleSource removes the source named by the action field. The cards
// showing its readings are cleared when they belong to it: their rows are gone
// from the source's point of view, and leaving them beside the deletion result
// would suggest the readings still describe a live feed.
func (a *App) deleteOracleSource() {
	id := strings.TrimSpace(a.oracle.fetchID)
	a.runWrite("Deleting source…", func() func() {
		err := a.services.DeleteOracleSource(id)
		return func() {
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
	})
}

func (a *App) setOracleEnabled(enabled bool) {
	id := strings.TrimSpace(a.oracle.fetchID)
	label := "Enabling source…"
	if !enabled {
		label = "Disabling source…"
	}
	a.runWrite(label, func() func() {
		err := a.services.SetOracleSourceEnabled(id, enabled)
		return func() {
			a.oracle.status = wrapErr(err)
			a.loadOracle()
		}
	})
}

func (a *App) fetchOracleSource() {
	sourceID := strings.TrimSpace(a.oracle.fetchID)
	a.runWrite("Fetching source…", func() func() {
		err := a.services.FetchOracleSource(sourceID)
		return func() {
			a.oracle.status = wrapErr(err)
			a.loadOracle()
			if err == nil {
				// A reading just stored must appear in the read cards already
				// showing that source; leaving the old value beside "Done"
				// reads as a fetch that stored nothing.
				if strings.TrimSpace(a.oracle.queryID) == sourceID &&
					(a.oracle.dataLoadedSource == sourceID || len(a.oracle.data) > 0 || a.oracle.dataErr != nil) {
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
	})
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
		names = append(names, tr("— choose a preset —"))
		for _, t := range a.oracle.templates {
			names = append(names, t.ID+" — "+t.Name)
		}
		ui.Field(c, tr("Preset (optional)"), func() {
			if ui.Combobox(c, &a.oracle.template, names).Changed() &&
				a.oracle.template != "" && a.oracle.template != names[0] {
				a.applyTemplate(strings.SplitN(a.oracle.template, " — ", 2)[0])
			}
		})
	}
	field(c, "Name", &a.oracle.addName)
	field(c, "URL", &a.oracle.addURL)
	ui.Collapsible(c, tr("Advanced"), &a.oracle.addAdvanced, func() {
		field(c, "Type (optional)", &a.oracle.addType)
		field(c, "Method (optional, defaults to GET)", &a.oracle.addMethod)
		field(c, "Path (optional JSON field)", &a.oracle.addPath)
		textAreaHint(c, "Request Headers (optional JSON object)",
			"Values are stored as request headers and shown as “configured”.", &a.oracle.addHeaders)
		ui.Field(c, tr("Interval (seconds)"), func() {
			ui.Column(c).Gap(2).Children(func() {
				ui.NumberInput(c, &a.oracle.addInterval, 0, maxOracleIntervalSeconds, 1)
				hint(c, "0 uses the default of 60; at most 30 days.")
			})
		})
	})
	ui.Row(c).Children(func() {
		if primaryButton(c, "Add Source").Disabled(!allSet(a.oracle.addName, a.oracle.addURL)).Clicked() {
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
		// Built-in presets carry no request headers. Keeping a private source's
		// Authorization value here would silently attach that secret to the
		// public endpoint the operator just chose.
		a.oracle.addHeaders = ""
		// A preset fills fields the disclosure hides; open it so the operator
		// can review the method, path and interval before Add Source.
		a.oracle.addAdvanced = true
		return
	}
}

func (a *App) addOracleSource() {
	input := SourceInput{
		Name:     strings.TrimSpace(a.oracle.addName),
		URL:      strings.TrimSpace(a.oracle.addURL),
		Type:     strings.TrimSpace(a.oracle.addType),
		Method:   strings.TrimSpace(a.oracle.addMethod),
		Path:     strings.TrimSpace(a.oracle.addPath),
		Headers:  strings.TrimSpace(a.oracle.addHeaders),
		Interval: int(a.oracle.addInterval),
	}
	a.runWrite("Adding source…", func() func() {
		src, err := a.services.AddOracleSource(input)
		return func() {
			if err != nil {
				a.oracle.status = errStatus(err)
				return
			}
			a.oracle.status = okStatus("Added %s (%s)", src.Name, src.ID)
			a.oracle.addName, a.oracle.addURL, a.oracle.addType = "", "", ""
			a.oracle.addMethod, a.oracle.addPath, a.oracle.addHeaders = "", "", ""
			a.oracle.addInterval = 0
			// The preset choice is part of the form, not a permanent mode: leaving
			// it selected would show a preset whose fields were just cleared, and
			// picking it again would not report a change.
			a.oracle.template = ""
			// A new source is newest-first, so try page zero. The loader commits
			// the offset only after a successful read; if it fails, the old page,
			// search and highlight remain the rows they actually describe.
			if a.loadOracleSourcePage(0) {
				// The new feed becomes the subject of the actions and read cards,
				// just as a chosen row does. Otherwise the list highlights it while
				// Fetch, Query and Latest still target the previous source.
				a.oracle.fetchID, a.oracle.queryID, a.oracle.latestID = src.ID, src.ID, src.ID
				a.oracle.selected = -1
				a.selectSourceByID(src.ID)
				a.refreshOracleReads()
			}
			a.loadOracleAuxiliary()
		}
	})
}

func (a *App) selectSourceByID(id string) {
	a.oracle.search, a.oracle.appliedSearch = "", ""
	for i, source := range a.filteredSources() {
		if source.ID == id {
			a.oracle.selected = i
			return
		}
	}
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
	// New scheduler rows carry the source name. The current page remains a
	// fallback for older stat providers and for rows loaded before a source was
	// renamed; an id neither source knows still identifies its row.
	names := make(map[string]string, len(a.oracle.sources))
	for _, source := range a.oracle.sources {
		names[source.ID] = source.Name
	}
	if stackedRecords(c) {
		a.oracleHealthRecords(c, health, names)
		return
	}
	ui.Table(c, &a.oracle.healthTable, []ui.TableColumn{
		{Title: tr("Source")},
		{Title: tr("Attempts"), MinWidth: tableColumnWidth(c, 90), Align: ui.End},
		{Title: tr("Successes"), MinWidth: tableColumnWidth(c, 90), Align: ui.End},
		{Title: tr("Failures"), MinWidth: tableColumnWidth(c, 90), Align: ui.End},
		{Title: tr("Last error")},
	}, len(health), func(row, col int) {
		h := health[row]
		switch col {
		case 0:
			tableText(c, oracleHealthName(h, names))
		case 1:
			tableText(c, strconv.Itoa(h.Attempts))
		case 2:
			tableText(c, strconv.Itoa(h.Successes))
		case 3:
			tableText(c, strconv.Itoa(h.Failures))
		case 4:
			if h.LastError == "" {
				tableText(c, tr("—"))
			} else {
				tableText(c, h.LastError)
			}
		}
	})
}

// oracleHealthRecords preserves every counter and the complete last error when
// the five-column health table no longer fits at enlarged text.
func (a *App) oracleHealthRecords(c *ui.Context, health []HealthRow, names map[string]string) {
	ui.Column(c).FillWidth().Gap(10).Children(func() {
		for _, h := range health {
			h := h
			name := oracleHealthName(h, names)
			recordPanel(c, func() {
				ui.Text(c, name).FontSize(textSize(c, 13)).Bold().
					TextColor(ui.Hex(ink)).SingleLine().Ellipsis("…").Tooltip(name)
				ui.Row(c).Wrap().Gap(18).Children(func() {
					healthMetric(c, "Attempts", h.Attempts, ink)
					healthMetric(c, "Successes", h.Successes, auroraGreen)
					tone := inkMuted
					if h.Failures > 0 {
						tone = auroraRed
					}
					healthMetric(c, "Failures", h.Failures, tone)
				})
				recordField(c, "Last error", valueOr(h.LastError))
			})
		}
	})
}

func healthMetric(c *ui.Context, label string, value int, tone string) {
	recordMetric(c, label, strconv.Itoa(value), tone)
}

func oracleHealthName(h HealthRow, names map[string]string) string {
	name := h.Name
	if name == "" {
		name = names[h.SourceID]
	}
	if name == "" {
		name = h.SourceID
	}
	return name
}

func (a *App) oracleRead(c *ui.Context) {
	formCard(c, "Query Data", "Read the most recent stored values for one source.", func() {
		a.oracleReadBody(c)
	})
	a.oracleLatest(c)
}

func (a *App) oracleReadBody(c *ui.Context) {
	field(c, "Source ID", &a.oracle.queryID)
	ui.Field(c, tr("Limit"), func() {
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
		if stackedRecords(c) {
			a.oracleReadRecords(c, data)
		} else {
			ui.Table(c, &a.oracle.dataTable, []ui.TableColumn{
				{Title: tr("Value")},
				{Title: tr("Block"), MinWidth: tableColumnWidth(c, 140), Align: ui.End},
			}, len(data), func(row, col int) {
				d := data[row]
				if col == 0 {
					tableText(c, d.Value)
				} else {
					tableText(c, strconv.FormatInt(d.BlockHeight, 10))
				}
			})
		}
	}
}

// oracleReadRecords gives stored values the full card width at enlarged text.
// The table's one-line cell reduced nested JSON and long observations to a
// prefix, even though the block and value read naturally as a stacked record.
func (a *App) oracleReadRecords(c *ui.Context, data []DataRow) {
	ui.Column(c).FillWidth().Gap(10).Children(func() {
		for _, reading := range data {
			reading := reading
			recordPanel(c, func() {
				recordMetric(c, "Block", strconv.FormatInt(reading.BlockHeight, 10), ink)
				recordField(c, "Value", reading.Value)
			})
		}
	})
}

// loadOpenReadings fills the Readings stage for the source already selected in
// the action field. It runs once per source/limit: a successful empty result or
// a failure is recorded in the same key, so opening the tab does not hammer the
// database every frame. The explicit Query/Get Latest buttons remain the retry.
func (a *App) loadOpenReadings() {
	if id := strings.TrimSpace(a.oracle.queryID); id != "" {
		limit := a.oracleQueryLimit()
		if a.oracle.dataLoadedSource != id || a.oracle.dataLoadedLimit != limit {
			a.clearOracleDataForAnotherRead(id, limit)
			_ = a.loadOracleData(id, limit)
		}
	}
	if id := strings.TrimSpace(a.oracle.latestID); id != "" && a.oracle.latestLoadedSource != id {
		a.clearOracleLatestForAnotherSource(id)
		_ = a.loadOracleLatest(id)
	}
}

// refreshOracleReads re-reads the Query and Latest cards that are already open
// (they hold rows or a failure). The key check invalidates a card whose source
// or limit changed, so a failed read cannot leave the previous answer under
// the new input.
func (a *App) refreshOracleReads() {
	// An empty successful read still means the card is open: a newly added
	// source usually has no values yet, and Refresh/live updates must pick up
	// the first row when the scheduler writes one. dataLoadedSource is the
	// open marker, not the row count.
	if a.oracle.dataLoadedSource != "" || len(a.oracle.data) > 0 || a.oracle.dataErr != nil {
		sourceID := strings.TrimSpace(a.oracle.queryID)
		limit := a.oracleQueryLimit()
		a.clearOracleDataForAnotherRead(sourceID, limit)
		_ = a.loadOracleData(sourceID, limit)
	}
	if a.oracle.latestLoadedSource != "" || a.oracle.latest != nil || a.oracle.latestErr != nil {
		sourceID := strings.TrimSpace(a.oracle.latestID)
		a.clearOracleLatestForAnotherSource(sourceID)
		_ = a.loadOracleLatest(sourceID)
	}
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

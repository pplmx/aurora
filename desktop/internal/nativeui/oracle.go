package nativeui

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// oracleState is the Oracle page's state.
type oracleState struct {
	// Add-source form.
	addName     string
	addURL      string
	addType     string
	addMethod   string
	addPath     string
	addHeaders  string
	addInterval string
	template    string

	// Per-source actions.
	fetchID  string
	queryID  string
	queryLim string
	latestID string

	sources   []SourceRow
	templates []TemplateRow
	health    []HealthRow
	data      []DataRow
	latest    *DataRow

	loadErr error

	// selected is the source row chosen, -1 for none.
	selected int

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
	sources, err := a.services.OracleSources()
	if err != nil {
		a.oracle.loadErr = err
		return
	}
	a.oracle.sources = sources
	if tpl, err := a.services.OracleTemplates(); err == nil {
		a.oracle.templates = tpl
	}
	if h, err := a.services.OracleHealth(); err == nil {
		a.oracle.health = h
	}
	a.oracle.loadErr = nil
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
	srcs := a.oracle.sources
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
			a.oracle.status = wrapErr(a.services.FetchOracleSource(strings.TrimSpace(a.oracle.fetchID)))
			a.loadOracle()
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
		a.oracle.status = wrapErr(a.services.DeleteOracleSource(strings.TrimSpace(a.oracle.fetchID)))
		a.loadOracle()
	}
}

func (a *App) oracleAdd(c *ui.Context) {
	formCard(c, "Add Source", "Point Aurora at an HTTP endpoint; the interval controls how often it is polled.", func() {
		a.oracleAddBody(c)
	})
}

func (a *App) oracleAddBody(c *ui.Context) {
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
	fieldHint(c, "Request Headers (optional JSON object)",
		"Values are stored as request headers and shown as “configured”.", &a.oracle.addHeaders)
	fieldHint(c, "Interval (seconds)", "0 uses the default of 60.", &a.oracle.addInterval)
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
		if t.Interval > 0 {
			a.oracle.addInterval = strconv.Itoa(t.Interval)
		}
		return
	}
}

func (a *App) addOracleSource() {
	interval := 60
	if s := strings.TrimSpace(a.oracle.addInterval); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			a.oracle.status = statusLine{text: "Interval must be zero or a positive number of seconds.", isErr: true}
			return
		}
		interval = n
	}
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
	a.oracle.addInterval = ""
	a.loadOracle()
}

func (a *App) oracleHealth(c *ui.Context) {
	formCard(c, "Feed Health", "Attempt, success and failure counters per source, with the last error kept verbatim.", func() {
		a.oracleHealthBody(c)
	})
}

func (a *App) oracleHealthBody(c *ui.Context) {
	if len(a.oracle.health) == 0 {
		emptyNote(c, "(no feed activity yet)")
		return
	}
	health := a.oracle.health
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
	field(c, "Limit", &a.oracle.queryLim)
	ui.Row(c).Children(func() {
		if secondaryButton(c, "Query").Clicked() {
			limit := 20
			if s := strings.TrimSpace(a.oracle.queryLim); s != "" {
				if n, err := strconv.Atoi(s); err == nil && n > 0 {
					limit = n
				}
			}
			rows, err := a.services.OracleData(strings.TrimSpace(a.oracle.queryID), limit)
			if err != nil {
				a.oracle.status = errStatus(err)
				a.oracle.data = nil
			} else {
				a.oracle.data = rows
				a.oracle.status = statusLine{}
			}
		}
	})
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
			d, err := a.services.OracleLatest(strings.TrimSpace(a.oracle.latestID))
			if err != nil {
				a.oracle.status = errStatus(err)
				a.oracle.latest = nil
			} else {
				a.oracle.latest = &d
				a.oracle.status = statusLine{}
			}
		}
	})
	if l := a.oracle.latest; l != nil {
		readout(c, []readoutRow{{"Value", l.Value}, {"Block", strconv.FormatInt(l.BlockHeight, 10)}})
	}
}

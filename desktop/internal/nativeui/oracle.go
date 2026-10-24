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
		ui.Column(c).FillWidth().Gap(16).Children(func() {
			a.oracleSources(c)
			a.oracleAdd(c)
			a.oracleHealth(c)
			a.oracleRead(c)
		})
	})
}

func (a *App) oracleSources(c *ui.Context) {
	sectionTitle(c, "Data Sources", "Every registered feed, its transport, and whether it is currently polled.")
	if a.oracle.loadErr != nil {
		ui.Text(c, "Failed to load sources: "+a.oracle.loadErr.Error()).FontSize(12).TextColor(ui.Hex("#c62828"))
		return
	}
	if len(a.oracle.sources) == 0 {
		emptyNote(c, "No sources yet.")
		return
	}
	srcs := a.oracle.sources
	ui.Table(c, &a.oracle.sourceTable, []ui.TableColumn{
		{Title: "Name"},
		{Title: "Type", Width: 100},
		{Title: "ID", Width: 140},
		{Title: "Status", Width: 90},
		{Title: "Headers", Width: 90},
	}, len(srcs), func(row, col int) {
		s := srcs[row]
		switch col {
		case 0:
			ui.Text(c, s.Name)
		case 1:
			ui.Text(c, s.Type)
		case 2:
			ui.Text(c, s.ID).SingleLine()
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

	// Per-source actions. A table row cannot host a button per cell without
	// making the whole grid interactive, so the id is entered once and the
	// actions apply to it — the same shape the Web UI's per-row buttons had,
	// with the selection made explicit.
	field(c, "Source ID for the actions below", &a.oracle.fetchID)
	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, "Enable").Clicked() {
			a.oracle.status = wrapErr(a.services.SetOracleSourceEnabled(strings.TrimSpace(a.oracle.fetchID), true))
			a.loadOracle()
		}
		if ui.Button(c, "Disable").Clicked() {
			a.oracle.status = wrapErr(a.services.SetOracleSourceEnabled(strings.TrimSpace(a.oracle.fetchID), false))
			a.loadOracle()
		}
		if ui.Button(c, "Fetch now").Clicked() {
			a.oracle.status = wrapErr(a.services.FetchOracleSource(strings.TrimSpace(a.oracle.fetchID)))
			a.loadOracle()
		}
		if ui.Button(c, "Delete").Clicked() {
			a.oracle.askDelete = true
		}
	})
	renderStatus(c, a.oracle.status)
	a.renderDeleteDialog(c)
}

func (a *App) renderDeleteDialog(c *ui.Context) {
	if !a.oracle.askDelete {
		return
	}
	switch ui.AlertDialog(c, &a.oracle.askDelete,
		"Delete this source?",
		"The source and its scheduling entry are removed. Readings already stored stay.",
		"Cancel", "Delete") {
	case 1:
		a.oracle.status = wrapErr(a.services.DeleteOracleSource(strings.TrimSpace(a.oracle.fetchID)))
		a.loadOracle()
	}
}

func (a *App) oracleAdd(c *ui.Context) {
	ui.Divider(c)
	sectionTitle(c, "Add Source", "Point Aurora at an HTTP endpoint; the interval controls how often it is polled.")
	if len(a.oracle.templates) > 0 {
		names := make([]string, 0, len(a.oracle.templates)+1)
		names = append(names, "— choose a preset —")
		for _, t := range a.oracle.templates {
			names = append(names, t.ID+" — "+t.Name)
		}
		chosen := ""
		if ui.Combobox(c, &chosen, names).Changed() && chosen != "" && chosen != names[0] {
			a.applyTemplate(strings.SplitN(chosen, " — ", 2)[0])
		}
	}
	field(c, "Name", &a.oracle.addName)
	field(c, "URL", &a.oracle.addURL)
	field(c, "Type (optional)", &a.oracle.addType)
	field(c, "Method (optional, defaults to GET)", &a.oracle.addMethod)
	field(c, "Path (optional JSON field)", &a.oracle.addPath)
	fieldHint(c, "Request Headers (optional JSON object)",
		"Values are stored as request headers and shown as “configured”.", &a.oracle.addHeaders)
	fieldHint(c, "Interval (seconds)", "0 uses the default of 60.", &a.oracle.addInterval)
	if ui.PrimaryButton(c, "Add Source").Clicked() {
		a.addOracleSource()
	}
}

func (a *App) applyTemplate(id string) {
	for _, t := range a.oracle.templates {
		if t.ID == id {
			a.oracle.addName = t.Name
			a.oracle.addType = t.ID
			return
		}
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
	ui.Divider(c)
	sectionTitle(c, "Feed Health", "Attempt, success and failure counters per source, with the last error kept verbatim.")
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
	ui.Divider(c)
	sectionTitle(c, "Query Data", "Read the most recent stored values for one source.")
	field(c, "Source ID", &a.oracle.queryID)
	field(c, "Limit", &a.oracle.queryLim)
	if ui.Button(c, "Query").Clicked() {
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
	renderStatus(c, a.oracle.status)
	if len(a.oracle.data) == 0 {
		return
	}
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

	ui.Divider(c)
	sectionTitle(c, "Latest Data", "The single most recent value recorded for a source.")
	field(c, "Source ID", &a.oracle.latestID)
	if ui.Button(c, "Get Latest").Clicked() {
		d, err := a.services.OracleLatest(strings.TrimSpace(a.oracle.latestID))
		if err != nil {
			a.oracle.status = errStatus(err)
			a.oracle.latest = nil
		} else {
			a.oracle.latest = &d
			a.oracle.status = statusLine{}
		}
	}
	if l := a.oracle.latest; l != nil {
		ui.Text(c, l.Value).FontSize(16).Bold()
	}
}

package nativeui

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

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
				ui.NumberInput(c, &a.oracle.addInterval, 0, maxOracleIntervalSeconds, 1).
					Description(tr("0 uses the default of 60; at most 30 days."))
				hint(c, "0 uses the default of 60; at most 30 days.")
			})
		})
	})
	ui.Row(c).Children(func() {
		if primaryButton(c, "Add Source", allSet(a.oracle.addName, a.oracle.addURL)).Clicked() {
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
	if rejectIncomplete(&a.oracle.status, "Source name and URL are required.", input.Name, input.URL) {
		return
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

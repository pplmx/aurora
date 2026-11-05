package nativeui

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *App) oracleSources(c *ui.Context) {
	formCard(c, "Data Sources", "Every registered feed, its transport, and whether it is currently polled.", func() {
		a.oracleSourcesBody(c)
	})
}

func (a *App) oracleSourcesBody(c *ui.Context) {
	if a.oracle.loadErr != nil && !loadWarning(c, "Failed to load sources", a.oracle.loadErr, len(a.oracle.sources) > 0) {
		return
	}
	// A new term is a new query over the whole collection, so it re-reads from the
	// first page. This runs before the empty-state branch, because a search that
	// matches nothing leaves the previous rows in place and would otherwise be
	// recorded as applied without ever being read.
	queryChanged := a.oracle.search != a.oracle.appliedSearch
	if queryChanged {
		// A write that clears the search has already chosen the row it created;
		// clearing the highlight here would throw that choice away in the same
		// frame that made it. So the selection is kept only when the field changed
		// from a term back to empty, which is what that write does.
		previousTerm := a.oracle.appliedSearch
		a.oracle.appliedSearch = a.oracle.search
		if previousTerm == "" || a.oracle.search != "" {
			a.oracle.selected = -1
		}
		a.loadOracleSourcePage(0)
	}
	if len(a.oracle.sources) == 0 {
		// An empty result is not always an empty store: a search over a store that
		// holds feeds needs "no match", because clearing the field is the fix and
		// the add-source guide's advice would be wrong.
		if strings.TrimSpace(a.oracle.appliedSearch) != "" {
			emptyNote(c, fmt.Sprintf(tr("No sources match “%s”."), strings.TrimSpace(a.oracle.appliedSearch)))
			return
		}
		emptyState(c, pageOracle, "No sources yet", "Add an HTTP feed in Add Source to start reading values.")
		return
	}
	previous, next := collectionToolbar(c, &a.oracle.search, "Search sources",
		"Search matches id, name, type or URL across the whole collection.",
		"Sources", a.oracle.offset, len(a.oracle.sources), a.oracle.hasMore)
	if previous || next {
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
	// The rows already answer the applied term: storage selected them.
	srcs := a.oracle.sources
	announceSearchResults(c, queryChanged, len(srcs))
	// A source's id is a timestamp plus a hash, so the record keeps it as a
	// field while the row itself is chosen by name. Choosing the record loads
	// the exact id into every action and read card below, which is how a
	// source reaches them without retyping an id nobody has in full.
	collectionList(c, "Data Sources",
		collectionRecords(srcs, func(source SourceRow) collectionRecord {
			return sourceCollectionRecord(c, source)
		}),
		a.oracle.selected, func(i int) { a.chooseOracleSource(srcs, i) })
}

// sourceCollectionRecord projects a registered feed onto a record. Enabled is a
// fact with a verdict hue because it is the state an operator scans for; the
// transport and header presence are facts, and the id is the field the
// actions copy.
func sourceCollectionRecord(c *ui.Context, source SourceRow) collectionRecord {
	name := strings.TrimSpace(source.Name)
	if name == "" {
		name = shortID(source.ID)
	}
	// A disabled feed keeps body ink; only "enabled" takes the verdict hue, so
	// the colour marks the state an operator is looking for rather than
	// painting every row.
	status := tr("disabled")
	var tone ui.Color
	if source.Enabled {
		status, tone = tr("enabled"), c.Theme().Success
	}
	headers := tr("—")
	if source.HasHeaders {
		headers = tr("configured")
	}
	return collectionRecord{
		title: name,
		facts: []collectionFact{
			// The type is a word ("custom", "price") except for the template that
			// names a format ("json"), which tr() leaves alone because it has no
			// entry.
			// The type is a word ("custom", "price") except for the template that
			// names a format ("json"), which tr() leaves alone because it has no
			// entry.
			{label: "Type", value: tr(valueOr(source.Type))},
			{label: "Status", value: status, tone: tone},
			{label: "Headers", value: headers},
		},
		fields: []collectionField{{label: "ID", value: source.ID}},
	}
}

// oracleSourceActions is the per-source workbench: the selected feed's
// endpoint and cadence, the id every action applies to, and the four controls.
// It lives under Add Source rather than under the table so the page's action
// column stays useful on the Health and Readings stages, and the table column
// is left to the collection.
func (a *App) oracleSourceActions(c *ui.Context) {
	formCard(c, "Source Actions", "Enable, disable, fetch or delete a feed; choose one in Sources.", func() {
		if rows := a.filteredSources(); a.oracle.selected >= 0 && a.oracle.selected < len(rows) {
			// The row is too narrow for the endpoint and the polling cadence,
			// the two facts that explain when and where the source fetches.
			chosen := rows[a.oracle.selected]
			copyRow(c, "URL", chosen.URL, nil)
			readout(c, []readoutRow{{"Every", sourceInterval(chosen.Interval)}})
		}
		if strings.TrimSpace(a.oracle.fetchID) == "" {
			fieldHint(c, "Source ID for the actions below",
				"Choose a source in Sources or enter its ID first.", &a.oracle.fetchID)
		} else {
			// The row above already names the chosen source; the first-step hint
			// beside a filled field would contradict it.
			field(c, "Source ID for the actions below", &a.oracle.fetchID)
		}
		copyRow(c, "Source ID", strings.TrimSpace(a.oracle.fetchID), nil)
		sourceReady := strings.TrimSpace(a.oracle.fetchID) != ""
		// A 2x2 grid rather than a wrapping row: the four labels have very
		// different widths, and at the workbench column's width the wrap left
		// Delete alone on a line that looked like a layout accident.
		actionPair := func(left, right func()) {
			ui.Row(c).FillWidth().Gap(space2).Children(func() {
				ui.Column(c).Grow(1).Shrink(1).Children(left)
				ui.Column(c).Grow(1).Shrink(1).Children(right)
			})
		}
		actionPair(
			func() {
				if secondaryButton(c, "Enable", sourceReady).FillWidth().Clicked() {
					a.setOracleEnabled(true)
				}
			},
			func() {
				if secondaryButton(c, "Disable", sourceReady).FillWidth().Clicked() {
					a.setOracleEnabled(false)
				}
			},
		)
		actionPair(
			func() {
				if secondaryButton(c, "Fetch now", sourceReady).FillWidth().Clicked() {
					a.fetchOracleSource()
				}
			},
			func() {
				deleteSource := dangerButton(c, "Delete", sourceReady).FillWidth().Bind(&a.oracle.deleteTrigger)
				if deleteSource.Clicked() {
					a.oracle.deleteName = a.resolveOracleSourceName(strings.TrimSpace(a.oracle.fetchID))
					a.oracle.askDelete = true
				}
			},
		)
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
	switch destructiveConfirm(c, &a.oracle.askDelete, "Delete this source?", detail, "Delete",
		a.oracle.deleteTrigger, a.oracle.deleteTrigger) {
	case destructiveConfirmed:
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
	if id == "" {
		return
	}
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
	if id == "" {
		return
	}
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
	if sourceID == "" {
		return
	}
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

package nativeui

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

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
		// An empty store is a missing feed, not a feed that has not been
		// polled yet: the panel names the step that creates the first row.
		note := "No poll attempts recorded yet."
		if a.oracle.loadErr == nil && len(a.oracle.sources) == 0 {
			note = "No feeds yet. Add a source to start polling."
		}
		emptyPanel(c, pageIcons[pageOracle], pageAccent(c, pageOracle), note)
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
	// Health lives in the half-width workbench column. Five columns cannot
	// keep "Attempts", "Successes", "Failures" and a verbatim error legible
	// there even at 1x — the headers ellipsised and the last error disappeared
	// first — so every window uses the record layout the enlarged-text path
	// already had.
	a.oracleHealthRecords(c, health, names)
}

// oracleHealthRecords draws the scheduler's per-source counters as records.
// Health lives in the half-width workbench column, where five columns could not
// keep "Attempts", "Successes", "Failures" and a verbatim error legible even at
// normal text.
func (a *App) oracleHealthRecords(c *ui.Context, health []HealthRow, names map[string]string) {
	collectionList(c, "Feed Health",
		collectionRecords(health, func(h HealthRow) collectionRecord {
			return healthCollectionRecord(c, h, names)
		}), -1, nil)
}

// healthCollectionRecord projects one source's scheduler counters onto a
// record. Failures take the error hue only when there are some: a zero painted
// red would read as a problem that is not there.
func healthCollectionRecord(c *ui.Context, h HealthRow, names map[string]string) collectionRecord {
	failureTone := ui.Color{}
	if h.Failures > 0 {
		failureTone = c.Theme().Danger
	}
	record := collectionRecord{
		title: oracleHealthName(h, names),
		facts: []collectionFact{
			{label: "Attempts", value: strconv.Itoa(h.Attempts)},
			{label: "Successes", value: strconv.Itoa(h.Successes), tone: c.Theme().Success},
			{label: "Failures", value: strconv.Itoa(h.Failures), tone: failureTone},
		},
	}
	if err := strings.TrimSpace(h.LastError); err != "" {
		record.fields = append(record.fields, collectionField{label: "Last error", value: err})
	}
	return record
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

package nativeui

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

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
		if secondaryButton(c, "Query").Disabled(!allSet(a.oracle.queryID)).Clicked() {
			a.readOracleData()
		}
	})
	if a.oracle.dataErr != nil &&
		!loadWarning(c, "Couldn't load readings", a.oracle.dataErr, len(a.oracle.data) > 0) {
		return
	}
	if len(a.oracle.data) > 0 {
		collectionList(c, "Oracle readings",
			collectionRecords(a.oracle.data, readingCollectionRecord), -1, nil)
	}
}

// readingCollectionRecord projects one stored observation onto a record. The
// value is the title because it is what the card is read for; the block that
// recorded it is the fact that proves when.
func readingCollectionRecord(reading DataRow) collectionRecord {
	return collectionRecord{
		title: valueOr(reading.Value),
		facts: []collectionFact{
			{label: "Block", value: strconv.FormatInt(reading.BlockHeight, 10)},
		},
		fields: []collectionField{{label: "Value", value: reading.Value}},
	}
}

func (a *App) readOracleData() {
	sourceID := strings.TrimSpace(a.oracle.queryID)
	if rejectIncomplete(&a.oracle.status, "Source ID is required for this read.", sourceID) {
		return
	}
	limit := a.oracleQueryLimit()
	a.clearOracleDataForAnotherRead(sourceID, limit)
	if err := a.loadOracleData(sourceID, limit); err != nil {
		a.oracle.status = errStatus(err)
	} else {
		a.oracle.status = statusLine{}
	}
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
		if secondaryButton(c, "Get Latest").Disabled(!allSet(a.oracle.latestID)).Clicked() {
			a.readOracleLatest()
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

func (a *App) readOracleLatest() {
	sourceID := strings.TrimSpace(a.oracle.latestID)
	if rejectIncomplete(&a.oracle.status, "Source ID is required for this read.", sourceID) {
		return
	}
	a.clearOracleLatestForAnotherSource(sourceID)
	if err := a.loadOracleLatest(sourceID); err != nil {
		a.oracle.status = errStatus(err)
	} else {
		a.oracle.status = statusLine{}
	}
}

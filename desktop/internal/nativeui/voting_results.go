package nativeui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// loadSessionResults reads a session's tally into the Results card. A failed
// read records the attempted session just like a successful empty read, so the
// automatic tab load does not retry a failing query on every frame; callers
// that change the subject clear the old rows first. The explicit Show Results
// and Refresh actions remain the retry path.
func (a *App) loadSessionResults(sessionID string) error {
	rows, err := a.services.SessionResults(sessionID)
	if err != nil {
		a.voting.resultsErr = err
		a.voting.resultsLoadedID = sessionID
		return err
	}
	a.voting.results = rows
	a.voting.resultsLoadedID = sessionID
	a.voting.resultsErr = nil
	return nil
}

// clearTallyForAnotherSession drops the rows when they describe a different
// session than the one about to be read, so a failed read cannot leave the
// previous session's numbers under the new id. Refreshing the same session
// keeps its rows and reports them as the last successful load.
func (a *App) clearTallyForAnotherSession(sessionID string) {
	if a.voting.resultsLoadedID == sessionID {
		return
	}
	a.voting.results = nil
	a.voting.resultsLoadedID = ""
	a.voting.resultsErr = nil
}

// sessionWindow renders a session's acceptance window as a compact pair of
// local times.
func sessionWindow(start, end int64) string {
	startLine, endLine := sessionWindowLines(start, end)
	if endLine == "" {
		return startLine
	}
	return startLine + " " + endLine
}

// sessionWindowLines splits the exact local start and end times so a table
// column can stack them without truncating either timestamp.
func sessionWindowLines(start, end int64) (string, string) {
	if start == 0 && end == 0 {
		return "—", ""
	}
	return time.Unix(start, 0).Format(localTimeLayout),
		"→ " + time.Unix(end, 0).Format(localTimeLayout)
}

func (a *App) votingResults(c *ui.Context) {
	// A tally belongs to a session; with none stored, the query field would ask
	// for an id that cannot exist yet and the tally card would stay empty.
	if a.votingHasNoSession(strings.TrimSpace(a.voting.resultsSession)) {
		a.votingNoSessionCard(c,
			"A tally belongs to a session; create one in Setup, then read its results here.")
		return
	}
	// The query and the tally are separate cards side by side: the session
	// field is one line of work, and the report it produces reads better at
	// half the window than as a full-width table under an empty field.
	twoColumns(c, func() {
		a.votingResultsQuery(c)
	}, func() {
		a.votingResultsTally(c)
	})
}

func (a *App) votingResultsQuery(c *ui.Context) {
	formCard(c, "Results", "Live tally per candidate for a session.", func() {
		field(c, "Session ID", &a.voting.resultsSession)
		ui.Row(c).Wrap().Gap(8).Children(func() {
			if secondaryButton(c, "Show Results", allSet(a.voting.resultsSession)).Clicked() {
				a.readSessionResults()
			}
		})
	})
}

func (a *App) votingResultsTally(c *ui.Context) {
	formCard(c, "Tally", "Votes per candidate in the loaded session.", func() {
		a.votingResultsBody(c)
	})
}

func (a *App) votingResultsBody(c *ui.Context) {
	id := strings.TrimSpace(a.voting.resultsSession)
	matches := id != "" && a.voting.resultsLoadedID == id
	if a.voting.resultsErr != nil && matches &&
		!loadWarning(c, "Failed to load results", a.voting.resultsErr, len(a.voting.results) > 0) {
		return
	}
	if !matches {
		// An empty field is a missing first step, not a finished empty read:
		// the session is chosen in Setup or pasted in, and only a session that
		// has not been read yet gets the "not loaded" line.
		note := "No tally loaded yet."
		if id == "" {
			note = "Choose a session in Setup or enter its ID first."
		}
		emptyPanel(c, pageIcons[pageVoting], pageAccent(c, pageVoting), note)
		if id == "" {
			a.resultsSessionShortcuts(c)
		}
		return
	}
	if len(a.voting.results) == 0 {
		emptyPanel(c, pageIcons[pageVoting], pageAccent(c, pageVoting), "No votes recorded for this session.")
		return
	}
	results := a.voting.results
	total := 0
	for _, r := range results {
		total += r.Votes
	}
	// The total is what a reader checks first ("did every ballot count?"); the
	// rows alone make them add the column up.
	readout(c, []readoutRow{{"Total votes", strconv.Itoa(total)}})
	if wideLayout(c) {
		a.votingResultBars(c, results)
	} else {
		// A bar cannot wrap a long name at enlarged text, so the tally falls
		// back to plain records: the same numbers, with the comparison made by
		// reading them instead of by the bar lengths.
		collectionList(c, "Voting results",
			collectionRecords(results, func(result ResultRow) collectionRecord {
				return collectionRecord{
					title: valueOr(result.Name),
					facts: []collectionFact{
						{label: "Votes", value: strconv.Itoa(result.Votes), tone: pageAccent(c, pageVoting)},
					},
				}
			}), -1, nil)
	}
}

// votingResultBars draws the tally as one row per candidate: the name and its
// exact count, over a bar scaled to the leader. The count stays exact, so the
// visual is a second reading of the same numbers rather than a replacement for
// them; the list role and each row's label keep the ordering and both facts
// available to a screen reader. At enlarged text, records take over because a
// bar cannot wrap a long name.
// resultsSessionShortcuts offers the newest stored sessions under the empty
// tally, so reading a session's results no longer requires leaving the tab for
// Setup just to copy an id. Setup keeps the full, searchable list.
func (a *App) resultsSessionShortcuts(c *ui.Context) {
	if len(a.voting.sessions) == 0 {
		return
	}
	recent := a.voting.sessions
	if len(recent) > 3 {
		recent = recent[:3]
	}
	items := make([]chipItem, 0, len(recent))
	for _, session := range recent {
		label := session.Title
		if label == "" {
			label = session.ID
		}
		items = append(items, chipItem{label: label, pick: func() { a.showResultsFor(session.ID) }})
	}
	ui.Column(c).FillWidth().AlignItems(ui.Center).Children(func() {
		chipPicker(c, "Sessions", items)
	})
}

// showResultsFor points the query, ballot and tally at one session and loads
// its tally — the same state a Setup selection leaves behind.
func (a *App) showResultsFor(id string) {
	a.voting.controlID, a.voting.voteSession, a.voting.resultsSession = id, id, id
	a.clearTallyForAnotherSession(id)
	_ = a.loadSessionResults(id)
	// The roster follows the same choice a Setup click records, so the ballot's
	// candidate names belong to the session the tally just read.
	a.restoreSessionChoice(id)
	a.loadSelectedSessionRoster()
}

func (a *App) votingResultBars(c *ui.Context, results []ResultRow) {
	leader := 1
	for _, result := range results {
		leader = max(leader, result.Votes)
	}
	list := ui.Column(c).FillWidth().Gap(space3).Role(ui.RoleList).
		Label(tr("Voting results"))
	list.Children(func() {
		for _, result := range results {
			result := result
			share := float32(result.Votes) / float32(leader) * 100
			ui.Column(c).FillWidth().Gap(space1).Role(ui.RoleListItem).
				Label(fmt.Sprintf(tr("%s: %d votes"), valueOr(result.Name), result.Votes)).
				Children(func() {
					ui.Row(c).FillWidth().Gap(space2).AlignItems(ui.Center).Children(func() {
						ui.Text(c, valueOr(result.Name)).FontSize(textSize(c, 12.5)).
							TextColor(c.Theme().Text).Grow(1).Shrink(1).MinWidth(0).
							SingleLine().Ellipsis("…").Tooltip(result.Name)
						ui.Text(c, strconv.Itoa(result.Votes)).FontSize(textSize(c, 12.5)).
							FontWeight(700).Font("monospace").FontFeatures("tnum").
							TextColor(pageAccent(c, pageVoting))
					})
					track := ui.Row(c).FillWidth().Height(6).Radius(3).
						Background(c.Theme().Background)
					track.Children(func() {
						if share <= 0 {
							return
						}
						ui.Row(c).WidthPercent(share).Height(6).Radius(3).
							Gradient(pageAccent(c, pageVoting), c.Theme().Accent, 90)
					})
				})
		}
	})
}

func (a *App) readSessionResults() {
	id := strings.TrimSpace(a.voting.resultsSession)
	if rejectIncomplete(&a.voting.status, "Session ID is required for this read.", id) {
		return
	}
	a.clearTallyForAnotherSession(id)
	if err := a.loadSessionResults(id); err != nil {
		a.voting.status = errStatus(err)
	} else {
		a.voting.status = statusLine{}
	}
}

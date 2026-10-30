package nativeui

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// applyVoteVoter moves the ballot identity and clears signing state that
// belonged to the previous voter. The private key is set separately by the
// registration path because that is the one response allowed to supply it.
func (a *App) applyVoteVoter(publicKey string) {
	publicKey = strings.TrimSpace(publicKey)
	if a.voting.voteVoterPub != publicKey {
		a.voting.votePriv = ""
		a.voting.voteResult = statusLine{}
	}
	a.voting.voteVoterPub = publicKey
	a.voting.voteSubjectPub = publicKey
	a.voting.voteSubjectKnown = true
}

// syncVoteSubject reacts to the public-key field being edited by hand. The
// first frame adopts state restored before the page was opened; after that,
// changing the visible voter invalidates the old private key and verdict.
func (a *App) syncVoteSubject() {
	current := strings.TrimSpace(a.voting.voteVoterPub)
	if !a.voting.voteSubjectKnown {
		a.voting.voteSubjectPub = current
		a.voting.voteSubjectKnown = true
		return
	}
	if current == a.voting.voteSubjectPub {
		return
	}
	a.voting.votePriv = ""
	a.voting.voteResult = statusLine{}
	a.voting.voteSubjectPub = current
}

func (a *App) votingVote(c *ui.Context) {
	// A ballot is keyed by a session id. With no session in the store the two
	// cards would ask for an id that cannot exist yet, so the tab names the
	// first move and links back instead.
	if a.votingHasNoSession(strings.TrimSpace(a.voting.voteSession), strings.TrimSpace(a.voting.controlID)) {
		a.votingNoSessionCard(c,
			"A ballot names a session; create one in Setup and its id fills this form.")
		return
	}
	// Cast Vote is the task and takes the leading column; the two session
	// lifecycle actions sit beside it rather than pushing the ballot form
	// below the fold on a default window.
	twoColumns(c, func() {
		a.votingCastVoteCard(c)
	}, func() {
		a.votingSessionControlsCard(c)
	})
}

// votingHasNoSession reports whether a stage keyed by a session id has nothing
// to work with: the store returned no sessions and the operator has not typed
// an id into any of the stage's own fields. A typed id keeps the stage open,
// because the session may live outside the page the list loaded.
func (a *App) votingHasNoSession(entered ...string) bool {
	if len(a.voting.sessions) > 0 {
		return false
	}
	for _, id := range entered {
		if id != "" {
			return false
		}
	}
	return true
}

// votingNoSessionCard is the first step the Ballot and Results tabs share when
// the store has no session yet: name what is missing and offer the stage that
// creates one.
func (a *App) votingNoSessionCard(c *ui.Context, body string) {
	card(c, func() {
		// The action sits on the sentence's line, right-aligned, so the card
		// reads as one call to the stage that creates a session. A narrow
		// window wraps the button under the text instead of compressing it.
		ui.Row(c).FillWidth().Wrap().Gap(space3).AlignItems(ui.Center).Children(func() {
			iconTile(c, pageIcons[pageVoting], pageAccent(c, pageVoting), iconTileMedium)
			ui.Column(c).Grow(1).Shrink(1).MinWidth(textSize(c, 220)).Gap(space1).Children(func() {
				heading(c, tr("No sessions yet."), sectionSize, 2)
				ui.Text(c, tr(body)).FontSize(textSize(c, 12)).TextColor(c.Theme().TextMuted)
			})
			if secondaryButton(c, "Open Setup").Clicked() {
				a.voting.tab = 0
			}
		})
	})
}

func (a *App) votingCastVoteCard(c *ui.Context) {
	formCard(c, "Cast Vote", "Sign a ballot for one candidate in one session.", func() {
		fieldPair(c, "Session ID", &a.voting.voteSession, "Candidate ID", &a.voting.voteCandidate)
		field(c, "Voter Public Key", &a.voting.voteVoterPub)
		field(c, "Voter Private Key", &a.voting.votePriv)
		a.useSavedKeyButton(c, a.voting.voteVoterPub, &a.voting.votePriv)
		ui.Row(c).Children(func() {
			if primaryButton(c, "Cast Vote", allSet(a.voting.voteSession, a.voting.voteCandidate, a.voting.voteVoterPub, a.voting.votePriv)).Clicked() {
				a.castVote()
			}
		})
		inlineStatus(c, a.voting.voteResult)
	})
}

func (a *App) votingSessionControlsCard(c *ui.Context) {
	formCard(c, "Session Controls", "Start or end a session by id.", func() {
		fieldHint(c, "Session ID",
			"Choose a session above or enter its ID first.", &a.voting.controlID)
		sessionReady := strings.TrimSpace(a.voting.controlID) != ""
		ui.Row(c).Wrap().Gap(8).Children(func() {
			if secondaryButton(c, "Start Session").Disabled(!sessionReady).Clicked() {
				a.setSessionControl("Starting session…", a.services.StartSession, "Session started.")
			}
			if secondaryButton(c, "End Session").Disabled(!sessionReady).Clicked() {
				a.setSessionControl("Ending session…", a.services.EndSession, "Session ended.")
			}
		})
		inlineStatus(c, a.voting.controlResult)
	})
}

func (a *App) setSessionControl(label string, fn func(string) error, success string) {
	id := strings.TrimSpace(a.voting.controlID)
	if id == "" {
		return
	}
	a.runWrite(label, func() func() {
		err := fn(id)
		return func() {
			if err != nil {
				a.voting.controlResult = errStatus(err)
				return
			}
			a.voting.controlResult = okStatus(success)
			a.loadVoting()
		}
	})
}

func (a *App) castVote() {
	a.voting.voteResult = statusLine{}
	if strings.TrimSpace(a.voting.voteSession) == "" || strings.TrimSpace(a.voting.voteCandidate) == "" {
		a.voting.voteResult = errorStatus("Choose a session and a candidate.")
		return
	}
	publicKey, privateKey := strings.TrimSpace(a.voting.voteVoterPub), strings.TrimSpace(a.voting.votePriv)
	sessionID, candidateID := strings.TrimSpace(a.voting.voteSession), strings.TrimSpace(a.voting.voteCandidate)
	if rejectIncomplete(&a.voting.voteResult, "Voter public key and private key are required.", publicKey, privateKey) {
		return
	}
	a.runWrite("Casting vote…", func() func() {
		err := a.services.CastVote(sessionID, candidateID, publicKey, privateKey)
		return func() {
			if err != nil {
				a.voting.voteResult = errStatus(err)
				return
			}
			a.rememberSuccessfulKey(publicKey, privateKey)
			a.voting.voteResult = okStatus("Ballot recorded.")
			a.voting.votePriv = ""
			a.loadVoting()
			// A tally already on screen for this session must include the ballot
			// that was just accepted; leaving the old numbers beside "Ballot
			// recorded" read as a vote that did not count. A failed refresh keeps
			// the old rows rather than replacing them with an error for a write
			// that succeeded.
			if sessionID != "" && sessionID == strings.TrimSpace(a.voting.resultsSession) &&
				(a.voting.resultsLoadedID == sessionID || len(a.voting.results) > 0 || a.voting.resultsErr != nil) {
				a.clearTallyForAnotherSession(sessionID)
				_ = a.loadSessionResults(sessionID)
			}
		}
	})
}

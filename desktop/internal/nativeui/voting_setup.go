package nativeui

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// votingView splits the page into the three stages an operator moves through:
// setup (register voters/candidates, create a session), ballot (start/end a
// session, cast a signed ballot), and results. The four areas used to be
// stacked at once, so a first-time operator had no visible order.
func (a *App) votingView(c *ui.Context) {
	a.syncVoteSubject()
	a.scrollPage(c, func() {
		// loadVoting keeps both lists when either read fails, so the warning
		// is drawn once for the page and both lists keep their last successful
		// rows. It used to be invisible entirely: nothing rendered loadErr.
		if a.voting.loadErr != nil {
			loadWarning(c, "Failed to load voting data", a.voting.loadErr,
				len(a.voting.voters) > 0 || len(a.voting.sessions) > 0 || len(a.voting.candidates) > 0)
		}
		// A store with nobody in it leads with the two registration forms; the
		// lists and the session form need a roster to mean anything. The guide
		// would repeat the first-record path, so it waits for the first record.
		first := a.votingNeedsFirstRecord()
		if !first {
			guide(c, &a.guides.voting,
				"Register voters and candidates in Setup. A voter registration returns the only copy of that voter's signing key.",
				"Create a session with a roster and a start/end window.",
				"Open Ballot to start the session and cast signed ballots.",
				"Open Results and select or enter the session to read the tally.",
			)
		}
		tabs(c, &a.voting.tab, "Voting sections", tr("Setup"), tr("Ballot"), tr("Results"))
		switch a.voting.tab {
		case 0:
			if first {
				twoColumns(c, func() {
					a.votingRegister(c)
				}, func() {
					firstRunPath(c, pageVoting, "No voters yet",
						"A session needs voters and at least one candidate; registering a voter returns the only copy of that voter's signing key.",
						"Fill in Register Voter and keep the private key it returns.",
						"Register at least one candidate for the ballot.",
						"Create a session with the roster and the window ballots are accepted in.",
					)
				})
				return
			}
			twoColumns(c, func() {
				a.votingVoterListCard(c)
				a.votingCandidateListCard(c)
				a.votingSessionListCard(c)
			}, func() {
				a.votingRegister(c)
				a.votingSessions(c)
			})
		case 1:
			a.votingVote(c)
		default:
			if id := strings.TrimSpace(a.voting.resultsSession); id != "" && a.voting.resultsLoadedID != id {
				// Opening Results with a session already chosen should show its
				// tally, not require another Show Results click. A failed or
				// empty load records the id, so it does not repeat every frame.
				a.clearTallyForAnotherSession(id)
				_ = a.loadSessionResults(id)
			}
			a.votingResults(c)
		}
	})
}

// votingNeedsFirstRecord is the Setup tab's first-write state: a successful
// read returned no voters, candidates or sessions, and nothing is filtered or
// selected. The registration forms lead because a session needs a roster, and
// the empty lists would only repeat what the forms do.
func (a *App) votingNeedsFirstRecord() bool {
	entered := a.voting.tab != 0 ||
		a.voting.voterSearch != "" || a.voting.voterApplied != "" ||
		a.voting.candSearch != "" || a.voting.candApplied != "" ||
		a.voting.sessionSearch != "" || a.voting.sessionApplied != "" ||
		a.voting.voterChoice >= 0 || a.voting.candChoice >= 0 || a.voting.sessionChoice >= 0
	return firstRecordOnly(a.voting.loadErr,
		len(a.voting.voters)+len(a.voting.candidates)+len(a.voting.sessions), entered)
}

func (a *App) votingRegister(c *ui.Context) {
	formCard(c, "Register Voter", "Returns the key pair the voter signs ballots with; keep the private half.", func() {
		validatedField(c, "Voter Name", &a.voting.voterName, &a.voting.voterNameErr)
		ui.Row(c).Children(func() {
			if primaryButton(c, "Register Voter", allSet(a.voting.voterName)).Clicked() {
				a.registerVoter()
			}
		})
		if a.voting.voterPriv != "" {
			ui.Text(c, tr("New voter key pair")).FontSize(textSize(c, 12)).Bold().TextColor(c.Theme().Text)
			copyRow(c, "Public key", a.voting.voterPub, func() {
				a.voting.status = okStatus("Public key copied.")
			})
			copyRow(c, "Private key", a.voting.voterPriv, func() {
				a.voting.status = okStatus("Private key copied. The store keeps only the public half; keep it safe.")
			})
			ui.Text(c, tr("The store keeps only the public half, so this is the only copy of the signing key.")).
				FontSize(textSize(c, 11)).TextColor(c.Theme().Warning)
		}
	})

	formCard(c, "Register Candidate", "Only the name is required; party and program are shown on the ballot when present.", func() {
		a.votingRegisterCandidateBody(c)
	})
}

func (a *App) votingRegisterCandidateBody(c *ui.Context) {
	validatedField(c, "Candidate Name", &a.voting.candName, &a.voting.candNameErr)
	fieldPair(c, "Party (optional)", &a.voting.candParty, "Program (optional)", &a.voting.candProgram)
	ui.Row(c).Children(func() {
		if secondaryButton(c, "Register Candidate", allSet(a.voting.candName)).Clicked() {
			a.registerCandidate()
		}
	})
	inlineStatus(c, a.voting.candResult)
	if a.voting.candCopyID != "" {
		copyRow(c, "Candidate ID", a.voting.candCopyID, func() {
			a.voting.status = okStatus("Candidate ID copied.")
		})
	}
}

// registerCandidate registers the candidate described by the card and keeps
// its id for the Copy button: the row list shortens ids, and the session
// roster field needs the full one.
func (a *App) registerCandidate() {
	a.voting.candNameErr = ""
	a.voting.candResult = statusLine{}
	a.voting.candCopyID = ""
	name := strings.TrimSpace(a.voting.candName)
	if name == "" {
		a.voting.candNameErr = "Candidate name is required."
		return
	}
	party, program := strings.TrimSpace(a.voting.candParty), strings.TrimSpace(a.voting.candProgram)
	a.runWrite("Registering candidate…", func() func() {
		cand, err := a.services.RegisterCandidate(name, party, program)
		return func() {
			if err != nil {
				a.voting.candResult = errStatus(err)
				return
			}
			a.voting.candResult = okStatus("Registered %s", cand.Name)
			a.voting.candCopyID = cand.ID
			a.voting.candName, a.voting.candParty, a.voting.candProgram = "", "", ""
			a.voting.candChoice = -1
			a.loadCandidatePage(0)
			// The new candidate is the context: highlight it, clear a search that
			// would hide it, and aim the ballot at it. The operator can still change
			// either choice before casting.
			a.selectCandidateByID(cand.ID)
			a.voting.voteCandidate = cand.ID
		}
	})
}

// selectCandidateByID highlights a candidate after registration.
func (a *App) selectCandidateByID(id string) {
	a.voting.candSearch, a.voting.candApplied = "", ""
	for i, candidate := range a.filteredCandidates() {
		if candidate.ID == id {
			a.voting.candChoice = i
			return
		}
	}
}

// registerVoter registers the named voter and carries the generated key pair
// into the ballot form.
func (a *App) registerVoter() {
	a.voting.voterNameErr = ""
	a.voting.status = statusLine{}
	name := strings.TrimSpace(a.voting.voterName)
	if name == "" {
		a.voting.voterNameErr = "Voter name is required."
		return
	}
	a.runWrite("Registering voter…", func() func() {
		v, err := a.services.RegisterVoter(name)
		return func() {
			if err != nil {
				a.voting.status = errStatus(err)
				return
			}
			a.applyVoterRegistration(v)
		}
	})
}

// applyVoterRegistration reports the new voter and fills the ballot form from
// it.
//
// The registration generates the ballot key pair and the store keeps only the
// public half, so this is the one chance to hand the private key to the
// operator. The pair is kept in the page state so the card can offer a Copy
// button for both halves, and the ballot fields are filled so the voter can
// sign straight away instead of copying a key that is no longer readable
// anywhere.
func (a *App) applyVoterRegistration(v VoterRow) {
	a.rememberSuccessfulKey(v.PublicKey, v.PrivateKey)
	a.voting.voterPub, a.voting.voterPriv = v.PublicKey, v.PrivateKey
	a.voting.status = okStatus("Registered %s. The ballot fields are filled from the new key pair.", v.Name)
	a.applyVoteVoter(v.PublicKey)
	a.voting.votePriv = v.PrivateKey
	a.voting.voterName = ""
	a.voting.voterSearch, a.voting.voterApplied = "", ""
	a.voting.voterChoice = -1
	if a.loadVoterPage(0) {
		a.selectVoterByPublicKey(v.PublicKey)
	}
}

func (a *App) selectVoterByPublicKey(publicKey string) {
	a.voting.voterSearch, a.voting.voterApplied = "", ""
	for i, voter := range a.filteredVoters() {
		if voter.PublicKey == publicKey {
			a.voting.voterChoice = i
			return
		}
	}
}

func (a *App) votingSessions(c *ui.Context) {
	formCard(c, "Create Session", "Choose the roster and the window during which ballots are accepted.", func() {
		a.votingCreateSessionBody(c)
	})
}

func (a *App) votingCreateSessionBody(c *ui.Context) {
	field(c, "Title", &a.voting.sessionTitle)
	field(c, "Description (optional)", &a.voting.sessionDesc)
	// The window is edited with the toolkit's own date and time controls:
	// asking an operator to type epoch seconds made the card unusable without
	// a clock and a converter outside the app. The two controls edit one
	// time.Time, shown in local time; the service still receives Unix seconds.
	var startDate, startTime ui.Element
	start := ui.Field(c, tr("Start"), func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				startDate = ui.DateInput(c, &a.voting.sessionStart).Label(tr("Start date")).
					Description(tr("Local time. Ballots are rejected before it."))
				startTime = ui.TimeInput(c, &a.voting.sessionStart).Label(tr("Start time")).
					Description(tr("Local time. Ballots are rejected before it."))
			})
			hint(c, "Local time. Ballots are rejected before it.")
		})
	})
	if startDate.Changed() || startTime.Changed() {
		a.voting.sessionStartErr = ""
	}
	attachFieldError(c, start, "Start", a.voting.sessionStartErr)
	if a.voting.sessionStartErr != "" {
		startTime.Error(localizeServiceErrorMessage(a.voting.sessionStartErr))
	}
	var endDate, endTime ui.Element
	end := ui.Field(c, tr("End"), func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				endDate = ui.DateInput(c, &a.voting.sessionEnd).Label(tr("End date")).
					Description(tr("Local time. Ballots are rejected after it."))
				endTime = ui.TimeInput(c, &a.voting.sessionEnd).Label(tr("End time")).
					Description(tr("Local time. Ballots are rejected after it."))
			})
			hint(c, "Local time. Ballots are rejected after it.")
		})
	})
	if endDate.Changed() || endTime.Changed() {
		a.voting.sessionEndErr = ""
	}
	attachFieldError(c, end, "End", a.voting.sessionEndErr)
	if a.voting.sessionEndErr != "" {
		endTime.Error(localizeServiceErrorMessage(a.voting.sessionEndErr))
	}
	fieldHint(c, "Candidate IDs (comma-separated)", "Leave empty to include every registered candidate.", &a.voting.sessionIDs)
	ui.Row(c).Children(func() {
		if primaryButton(c, "Create Session", allSet(a.voting.sessionTitle)).Clicked() {
			a.createVotingSession()
		}
	})
	inlineStatus(c, a.voting.sessionResult)
	if a.voting.sessionCopyID != "" {
		copyRow(c, "Session ID", a.voting.sessionCopyID, func() {
			a.voting.status = okStatus("Session ID copied.")
		})
	}
}

// addCandidateToRoster appends the chosen candidate to the session form's
// roster field. The field takes comma-separated ids, so building a custom
// roster used to mean copying UUIDs by hand; this is the button that makes the
// form usable for anything but "every candidate".
func (a *App) addCandidateToRoster() {
	candidates := a.filteredCandidates()
	if a.voting.candChoice < 0 || a.voting.candChoice >= len(candidates) {
		a.voting.status = errorStatus("Choose a candidate first.")
		return
	}
	candidate := candidates[a.voting.candChoice]
	ids := splitParticipants(a.voting.sessionIDs)
	for _, id := range ids {
		if id == candidate.ID {
			a.voting.status = okStatus("%s is already in the session roster.", candidate.Name)
			return
		}
	}
	ids = append(ids, candidate.ID)
	a.voting.sessionIDs = strings.Join(ids, ",")
	a.voting.status = okStatus("Added %s to the session roster.", candidate.Name)
}

func (a *App) createVotingSession() {
	a.voting.sessionResult = statusLine{}
	a.voting.sessionStartErr, a.voting.sessionEndErr = "", ""
	a.voting.sessionCopyID = ""
	start := a.voting.sessionStart.Unix()
	end := a.voting.sessionEnd.Unix()
	if a.voting.sessionStart.IsZero() {
		a.voting.sessionStartErr = "Choose a start time."
	}
	if a.voting.sessionEnd.IsZero() {
		a.voting.sessionEndErr = "Choose an end time."
	}
	if a.voting.sessionStartErr != "" || a.voting.sessionEndErr != "" {
		return
	}
	if end <= start {
		a.voting.sessionEndErr = "End time must be after the start time."
		return
	}
	title := strings.TrimSpace(a.voting.sessionTitle)
	if rejectIncomplete(&a.voting.sessionResult, "Session title is required.", title) {
		return
	}
	ids := splitParticipants(a.voting.sessionIDs)
	if len(ids) == 0 {
		// The visible list is one page, but an empty roster means every
		// registered candidate. Read the complete id set only for this write;
		// the collection itself remains paged.
		all, err := a.services.Candidates(0, 0)
		if err != nil {
			a.voting.sessionResult = errStatus(err)
			return
		}
		for _, c := range all {
			ids = append(ids, c.ID)
		}
	}
	if len(ids) == 0 {
		a.voting.sessionResult = errorStatus("Register at least one candidate before creating a session.")
		return
	}
	description := strings.TrimSpace(a.voting.sessionDesc)
	a.runWrite("Creating session…", func() func() {
		s, err := a.services.CreateSession(title, description, ids, start, end)
		return func() {
			if err != nil {
				a.voting.sessionResult = errStatus(err)
				return
			}
			a.voting.sessionResult = okStatus("Created session %s", s.Title)
			a.voting.sessionCopyID = s.ID
			a.voting.sessionTitle, a.voting.sessionDesc, a.voting.sessionIDs = "", "", ""
			a.voting.sessionStart, a.voting.sessionEnd = defaultSessionWindow(time.Now())
			a.voting.sessionChoice = -1
			a.loadSessionPage(0)
			// The new session becomes the context for controls, ballot and tally;
			// highlight its row so a successful create has a visible result even
			// before the copy button is used.
			a.selectSessionByID(s.ID)
			a.clearTallyForAnotherSession(s.ID)
			a.voting.controlID, a.voting.voteSession, a.voting.resultsSession = s.ID, s.ID, s.ID
		}
	})
}

// selectSessionByID highlights a session after creation. The search is
// cleared first so the index is in the list the table actually shows.
func (a *App) selectSessionByID(id string) {
	a.voting.sessionSearch, a.voting.sessionApplied = "", ""
	a.clearSessionRoster()
	for i, session := range a.filteredSessions() {
		if session.ID == id {
			a.voting.sessionChoice = i
			a.loadSelectedSessionRoster()
			return
		}
	}
}

// defaultSessionWindow is the window the Create Session card starts with: now
// (to the minute) until a day later, so a session is creatable without
// touching the date and time controls.
func defaultSessionWindow(now time.Time) (start, end time.Time) {
	start = now.Truncate(time.Minute)
	return start, start.Add(24 * time.Hour)
}

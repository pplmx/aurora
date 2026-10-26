package nativeui

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// tokenRecorder is a fakeServices that remembers which account each write
// form sent, so a test can prove the page reads the input the operator filled
// in rather than a leftover from another section.
type tokenRecorder struct {
	*fakeServices
	approveOwner   string
	approveSpender string
	fromOwner      string
	fromTo         string
	allowOwner     string
	allowSpender   string
}

func (r *tokenRecorder) ApproveToken(tokenID, owner, spender, amount, priv string) error {
	r.approveOwner, r.approveSpender = owner, spender
	return nil
}

func (r *tokenRecorder) TransferFromToken(tokenID, owner, to, amount, spenderPriv string) error {
	r.fromOwner, r.fromTo = owner, to
	return nil
}

func (r *tokenRecorder) TokenAllowance(tokenID, owner, spender string) (string, error) {
	r.allowOwner, r.allowSpender = owner, spender
	return "10", nil
}

// TestTokenWriteFormsUseTheirOwnInputs pins that Approve, Transfer From and
// Allowance each read the fields they display. Approve and Allowance used to
// have no owner input at all: they silently sent the Transfer section's "From"
// address, so a spender granted from the Approve form was signed for whichever
// account the operator had typed into a different form — and the section gave
// no way to see or change it. Transfer From borrowed Transfer's recipient the
// same way.
//
// The handlers are driven directly: a click on a section title and a click on
// the button below it are different controls, and the Tester resolves a label
// by text, so it would hit whichever one it found first.
func TestTokenWriteFormsUseTheirOwnInputs(t *testing.T) {
	rec := &tokenRecorder{fakeServices: &fakeServices{}}
	app := New(rec)

	// A leftover Transfer address must never be what another form signs for.
	app.token.tokenID = "t1"
	app.token.xFrom = "transfer-from"
	app.token.xTo = "transfer-to"

	app.token.approveOwner = "approve-owner"
	app.token.approveTo = "approve-spender"
	app.token.approveAmt = "5"
	app.token.approvePriv = "priv"
	app.approveToken()
	require.Equal(t, "approve-owner", rec.approveOwner,
		"Approve must sign as the owner named in the Approve form")
	require.Equal(t, "approve-spender", rec.approveSpender)

	app.token.fromOwner = "allowance-source"
	app.token.fromTo = "from-to"
	app.token.fromAmount = "3"
	app.token.fromPriv = "priv"
	app.transferFromToken()
	require.Equal(t, "allowance-source", rec.fromOwner)
	require.Equal(t, "from-to", rec.fromTo,
		"Transfer From must send the recipient named in its own form")

	app.token.allowanceOwner = "allow-owner"
	app.token.allowanceTo = "allow-spender"
	app.loadAllowance()
	require.Equal(t, "allow-owner", rec.allowOwner,
		"Allowance must read the owner named in the Allowance form")
	require.Equal(t, "allow-spender", rec.allowSpender)
}

// TestTokenFormsRenderTheirOwnerInputs pins the fields the sections show, so a
// form cannot silently lose the account it acts on again.
func TestTokenFormsRenderTheirOwnerInputs(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 4200)
	app.ShowPage("Token")
	tester.Frame()

	texts := tester.Texts()
	for _, want := range []string{
		"Owner (base64 public key)", // Approve, Transfer From and Allowance
		"Spender (base64 public key)",
	} {
		count := 0
		for _, got := range texts {
			if got == want {
				count++
			}
		}
		require.Positive(t, count, "the Token page must show a %q input", want)
	}
}

// voterKeyRecorder returns a registration that carries a private key, the way
// the real service does.
type voterKeyRecorder struct {
	*fakeServices
}

func (r *voterKeyRecorder) RegisterVoter(name string) (VoterRow, error) {
	return VoterRow{PublicKey: "pk-" + name, Name: name, PrivateKey: "sk-" + name}, nil
}

// TestRegisterVoterKeepsTheSigningKey covers the one-time key handover. The
// registration generates the ballot key pair and stores only the public half,
// so a window that dropped the private key left the operator with a voter that
// could never sign a ballot — and nothing else can produce that key again.
func TestRegisterVoterKeepsTheSigningKey(t *testing.T) {
	app := New(&voterKeyRecorder{fakeServices: &fakeServices{}})
	app.voting.voterName = "  Alice  "

	app.registerVoter()

	require.Empty(t, app.voting.voterNameErr)
	require.Equal(t, "pk-Alice", app.voting.voteVoterPub)
	require.Equal(t, "sk-Alice", app.voting.votePriv,
		"the ballot form must receive the signing key the registration returned")
	require.Equal(t, "sk-Alice", app.voting.voterPriv,
		"the page must keep the key pair for its copy buttons: it is the only copy")
	require.Contains(t, app.voting.status.text, "Registered Alice")
	require.Empty(t, app.voting.voterName, "a successful registration clears the form")
}

// TestRegisterVoterRejectsAnEmptyName keeps the guard in front of the service,
// where it names the field instead of surfacing an opaque error.
func TestRegisterVoterRejectsAnEmptyName(t *testing.T) {
	app := New(&voterKeyRecorder{fakeServices: &fakeServices{}})
	app.voting.voterName = "   "

	app.registerVoter()

	require.NotEmpty(t, app.voting.voterNameErr)
	require.Empty(t, app.voting.votePriv, "a rejected registration must not fill the ballot key")
}

// TestOracleLatestIsReachableWithoutData pins that the single-value read is
// always on screen. It used to be the tail of Query Data, which returned early
// when a query found no rows — so on a database with no stored readings the
// "Get Latest" control did not exist at all.
func TestOracleLatestIsReachableWithoutData(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 2000)
	app.ShowPage("Oracle")
	tester.Frame()

	require.True(t, tester.HasText("Query Data"))
	require.True(t, tester.HasText("Latest Data"),
		"the latest-value section must render even when no readings are loaded")
	require.True(t, tester.HasText("Get Latest"), "its button must be on screen")
}

// TestTokenWritesNameTheirSubject pins the one thing a write card cannot show
// by itself: which token it acts on. All the write cards use the same token
// id, set by Create or Inspect, and the page used to give no sign of it.
func TestTokenWritesNameTheirSubject(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Token")
	tester.Frame()
	require.True(t, tester.HasText("No token selected"),
		"an empty selection must be stated, not implied")

	app.token.tokenID = "AUR"
	tester.Frame()
	require.True(t, tester.HasText("Acting on"))
	require.True(t, tester.HasText("AUR"))
}

// TestSessionListShowsTheBallotWindow pins that a listed session carries the
// window a ballot is accepted in. The list used to show id, title and status
// only, so the one fact that explains a refused vote was missing until the
// voter tried and failed.
func TestSessionListShowsTheBallotWindow(t *testing.T) {
	start := time.Date(2026, 10, 25, 9, 0, 0, 0, time.Local)
	end := start.Add(2 * time.Hour)
	fake := &fakeServices{sessions: []SessionRow{{
		ID: "s1", Title: "Budget", Status: "active",
		StartTime: start.Unix(), EndTime: end.Unix(),
	}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 2000)
	app.ShowPage("Voting")
	tester.Frame()

	require.True(t, tester.HasText("Window"), "the column must be labelled")
	require.True(t, tester.HasText(sessionWindow(start.Unix(), end.Unix())),
		"the session's acceptance window must be on screen")
}

// TestOraclePresetKeepsItsChoice pins that the preset box holds what the
// operator picked. The selection lived in a local variable, which the next
// frame reset, so the box always read empty and the caption was missing
// entirely.
func TestOraclePresetKeepsItsChoice(t *testing.T) {
	fake := &fakeServices{}
	fake.templates = []TemplateRow{{
		ID: "coingecko", Name: "CoinGecko price",
		URL:  "https://api.example.com/price",
		Path: "bitcoin.usd", Method: "GET", Interval: 60,
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 2400)
	app.ShowPage("Oracle")
	tester.Frame()

	require.True(t, tester.HasText("Preset (optional)"), "the box needs a caption")

	// Choosing a preset fills the form from it and is remembered.
	app.oracle.template = "coingecko — CoinGecko price"
	app.applyTemplate("coingecko")
	tester.Frame()
	require.Equal(t, "CoinGecko price", app.oracle.addName)
	require.Equal(t, "coingecko", app.oracle.addType)
	require.Equal(t, "coingecko — CoinGecko price", app.oracle.template,
		"the choice must survive the next frame")
}

// TestChoosingAPresetFillsEveryField pins the whole preset, not just its name.
// Filling only the name left the URL empty, so Add Source refused a preset the
// CLI adds without complaint — the two surfaces disagreed about what a preset
// is.
func TestChoosingAPresetFillsEveryField(t *testing.T) {
	fake := &fakeServices{templates: []TemplateRow{{
		ID: "btc-price", Name: "Bitcoin Price",
		URL:    "https://api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=usd",
		Method: "GET", Path: "bitcoin.usd", Interval: 60,
	}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 2400)
	app.ShowPage("Oracle")
	tester.Frame()

	app.applyTemplate("btc-price")

	require.Equal(t, "Bitcoin Price", app.oracle.addName)
	require.Equal(t, "btc-price", app.oracle.addType)
	require.Equal(t, "https://api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=usd",
		app.oracle.addURL, "the preset's URL must reach the form")
	require.Equal(t, "GET", app.oracle.addMethod)
	require.Equal(t, "bitcoin.usd", app.oracle.addPath)
	require.Equal(t, "60", app.oracle.addInterval)

	// And the filled form is addable: no field the service needs is blank.
	require.NotEmpty(t, app.oracle.addURL, "Add Source needs a URL")
}

// TestShortIDKeepsACopyablePrefix pins the table-cell shortening: long ids keep
// enough of themselves to recognise and paste, and are marked when cut.
func TestShortIDKeepsACopyablePrefix(t *testing.T) {
	require.Equal(t, "abc", shortID("abc"), "a short id is left alone")
	require.Equal(t, "draw-1", shortID("draw-1"))

	long := "20261025233845.347259429-29ee362515b7c1fc06313345ef6be416"
	got := shortID(long)
	require.Equal(t, "202610252338…", got)
	require.True(t, strings.HasPrefix(long, strings.TrimSuffix(got, "…")),
		"the shortened id must be a prefix of the real one")
}

// TestChoosingAHistoryRowFillsTheVerifier pins the path from a shortened id in
// the table to the full one the verifier needs: a row click loads the draw's
// real id, which is the only place the table's ellipsis does not cut.
func TestChoosingAHistoryRowFillsTheVerifier(t *testing.T) {
	const full = "54c95915ba8b8f3e"
	fake := &fakeServices{draws: []DrawRow{{ID: full, BlockHeight: 2, Winners: []string{"Dan"}}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 2000)
	app.ShowPage("Lottery")
	tester.Frame()

	require.Equal(t, "54c95915ba8b…", shortID(full), "the table shows a shortened id")
	require.Empty(t, app.lottery.verifyID)

	require.NoError(t, tester.Click(shortID(full)), "click the row's id cell")
	tester.Frame()
	require.Equal(t, full, app.lottery.verifyID,
		"choosing the row must load the full id into the verifier")
}

// TestTokenListFillsTheSubject pins the page's entry point: the stored tokens
// are listed, and choosing one makes it the token every other card acts on.
// Before this the page could only work with an id the operator already had,
// which nothing in the window showed.
func TestTokenListFillsTheSubject(t *testing.T) {
	fake := &fakeServices{tokenRows: []TokenRow{
		{ID: "AUR", Name: "Aurora Coin", Symbol: "AUR", TotalSupply: "1000"},
		{ID: "GOLD", Name: "Gold", Symbol: "AU", TotalSupply: "50"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("Token")
	tester.Frame()

	require.True(t, tester.HasText("Aurora Coin"), "the stored tokens must be listed")
	require.True(t, tester.HasText("Gold"))
	require.Empty(t, app.token.tokenID, "nothing is chosen until the operator picks")

	require.NoError(t, tester.Click("AUR"))
	tester.Frame()
	require.Equal(t, "AUR", app.token.tokenID,
		"choosing a token must make it the subject of the write cards")
}

// TestNFTCollectionLoadsOnOpen pins the page's entry point: the whole
// collection is listed without asking for an owner key, so an asset minted in
// another session is reachable.
func TestNFTCollectionLoadsOnOpen(t *testing.T) {
	fake := &fakeServices{nftRows: []NFTRow{
		{ID: "n1", Name: "First Art", Owner: "owner-a", Creator: "owner-a", BlockHeight: 5},
		{ID: "n2", Name: "Second Art", Owner: "owner-b", Creator: "owner-b", BlockHeight: 6},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("NFT")
	tester.Frame()

	require.True(t, tester.HasText("First Art"), "the collection must load with the page")
	require.True(t, tester.HasText("Second Art"))
	require.Len(t, app.nft.list, 2)
}

// TestChoosingAnNFTRowFillsEveryCard pins the path from a shortened id in the
// collection to the full one: choosing a row fills Inspect, Transfer, Burn and
// History with that asset, so no card asks for an id the table cut.
func TestChoosingAnNFTRowFillsEveryCard(t *testing.T) {
	const full = "5c43a418-ca80-46ba-af63-cc7893d1a5e3"
	fake := &fakeServices{nftRows: []NFTRow{{ID: full, Name: "First Art", BlockHeight: 5}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("NFT")
	tester.Frame()

	require.NoError(t, tester.Click(shortID(full)), "click the row's id cell")
	tester.Frame()
	require.Equal(t, full, app.nft.id, "Inspect must receive the asset")
	require.Equal(t, full, app.nft.transferID, "Transfer must receive the asset")
	require.Equal(t, full, app.nft.burnID, "Burn must receive the asset")
	require.Equal(t, full, app.nft.historyID, "History must receive the asset")
}

// TestTableRowsKeepTheirHeight pins the rule the clipped rows taught: a table
// row is as high as its tallest cell, and a wrapped cell is cut off at that
// height. With every free-text cell on one line, a row showing a long value is
// exactly as high as a row showing a short one.
func TestTableRowsKeepTheirHeight(t *testing.T) {
	long := "A very long participant list that would certainly wrap onto a second line if it were allowed to"
	row := func(participants []string) float32 {
		fake := &fakeServices{draws: []DrawRow{{
			ID: "aaaa1111", BlockHeight: 1, Participants: participants, Winners: participants[:1],
		}}}
		app := New(fake)
		// A narrow window: with room to spare nothing wraps, so the row height
		// only tells the two cases apart where the column is squeezed.
		tester := ui.NewTester(app.View, 900, 2000)
		app.ShowPage("Lottery")
		tester.Frame()
		r, ok := tester.Find(shortID("aaaa1111"))
		require.True(t, ok)
		return r.H
	}

	require.Equal(t, row([]string{"Bob"}), row([]string{long, "Bob"}),
		"a long value must not make the row taller: the cell is one line")
}

// TestChoosingAnOracleRowFillsTheActions pins that a source's id reaches the
// action buttons and the read cards. The table shows a shortened id, so
// choosing the row is the only way to aim an action at it.
func TestChoosingAnOracleRowFillsTheActions(t *testing.T) {
	const full = "20261025233845.347259429-29ee362515b7c1fc06313345ef6be416"
	fake := &fakeServices{}
	fake.sources = []SourceRow{{ID: full, Name: "btc", Enabled: true}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 2400)
	app.ShowPage("Oracle")
	tester.Frame()

	require.NoError(t, tester.Click(shortID(full)), "click the row's id cell")
	tester.Frame()
	require.Equal(t, full, app.oracle.fetchID, "the actions must receive the source")
	require.Equal(t, full, app.oracle.queryID, "the query card must receive it")
	require.Equal(t, full, app.oracle.latestID, "the latest card must receive it")
}

// TestLedgerListsRecordedBlocks pins that the ledger page shows what the chain
// recorded, not only how many blocks there are: an audit trail nobody can read
// is not an audit trail.
func TestLedgerListsRecordedBlocks(t *testing.T) {
	fake := &fakeServices{blocks: []BlockRow{
		{Height: 2, Module: "token", Summary: `{"module":"token","amount":"100"}`},
		{Height: 1, Module: "lottery", Summary: `{"module":"lottery","seed":"abc"}`},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 2000)
	app.ShowPage("Blockchain")
	tester.Frame()

	require.True(t, tester.HasText("Height"))
	require.True(t, tester.HasText("token"), "a recorded module must be listed")
	require.True(t, tester.HasText("lottery"))
	require.Contains(t, joinedTexts(tester), `"amount":"100"`)
}

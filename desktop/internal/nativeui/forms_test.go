package nativeui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/domain/token"
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
	app := New(workbenchFakes())
	tester := ui.NewTester(app.View, 1280, 4200)
	app.ShowPage("Token")
	tester.Frame()

	for _, want := range []string{"To (base64 public key)", "Amount", "Owner Private Key"} {
		require.Truef(t, tester.HasText(want), "the default Mint form must show %q", want)
	}
	for _, tc := range []struct {
		tab  string
		want []string
	}{
		{"Transfer", []string{"From (base64 public key)", "To (base64 public key)", "From Private Key"}},
		{"Approve", []string{"Owner (base64 public key)", "Spender (base64 public key)", "Owner Private Key"}},
		{"Transfer From", []string{"Owner (base64 public key)", "To (base64 public key)", "Spender Private Key"}},
		{"Burn", []string{"From (base64 public key)", "Amount", "Owner Private Key"}},
	} {
		require.NoErrorf(t, tester.Click(tc.tab), "select the %s operation", tc.tab)
		tester.Frame()
		for _, want := range tc.want {
			require.Truef(t, tester.HasText(want), "the %s form must show %q", tc.tab, want)
		}
	}
	for _, want := range []string{"Allowance", "Get Allowance"} {
		require.Truef(t, tester.HasText(want),
			"the read-only allowance card must stay visible outside the write tabs")
	}
}

// TestTokenAllowanceIsOutsideTheWriteTabs keeps the read-only allowance query
// in the left column. Putting it sixth in the write tab row pushed "Allowance"
// past the right edge of its card at the normal desktop width.
func TestTokenAllowanceIsOutsideTheWriteTabs(t *testing.T) {
	app := New(workbenchFakes())
	tester := ui.NewTester(app.View, 1280, 2200)
	app.ShowPage("Token")
	tester.Frame()

	burn, ok := tester.Find("Burn")
	require.True(t, ok)
	allowance, ok := tester.Find("Allowance")
	require.True(t, ok)
	require.Less(t, allowance.X, burn.X, "Allowance belongs in the read column")
	require.Greater(t, allowance.Y, burn.Y, "Allowance must not share the write-tab row")
	require.True(t, tester.HasText("Get Allowance"))
}

// TestTokenDecimalsErrorStaysWithItsField opens the advanced section and marks
// the number input, rather than hiding the only explanation inside a collapsed
// section or leaving it in the page-level status strip.
func TestTokenDecimalsErrorStaysWithItsField(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 3200)
	app.ShowPage("Token")
	tester.Frame()

	app.token.name = "Gold"
	app.token.symbol = "GLD"
	app.token.supply = "1000"
	app.token.owner = "owner-pk"
	app.token.decimals = 128
	app.createToken()
	require.NotEmpty(t, app.token.decimalsErr)
	require.True(t, app.token.createAdvanced, "the invalid field must become visible")
	require.Empty(t, app.token.tokenID, "the invalid value must not reach the service")
	require.True(t, app.token.status.empty())

	tester.Frame()
	require.True(t, tester.HasText("Decimals must be a whole number between 0 and 127."))
}

// voterKeyRecorder returns a registration that carries a private key, the way
// the real service does.
type voterKeyRecorder struct {
	*fakeServices
}

func (r *voterKeyRecorder) RegisterVoter(name string) (VoterRow, error) {
	voter := VoterRow{PublicKey: "pk-" + name, Name: name, PrivateKey: "sk-" + name}
	r.voterRows = append([]VoterRow{voter}, r.voterRows...)
	return voter, nil
}

// TestRegisterVoterKeepsTheSigningKey covers the one-time key handover. The
// registration generates the ballot key pair and stores only the public half,
// so a window that dropped the private key left the operator with a voter that
// could never sign a ballot — and nothing else can produce that key again.
func TestRegisterVoterKeepsTheSigningKey(t *testing.T) {
	app := New(&voterKeyRecorder{fakeServices: &fakeServices{}})
	app.voting.voterName = "  Alice  "
	app.voting.voterSearch, app.voting.voterApplied = "old", "old"

	app.registerVoter()

	require.Empty(t, app.voting.voterNameErr)
	require.Equal(t, "pk-Alice", app.voting.voteVoterPub)
	require.Equal(t, "sk-Alice", app.voting.votePriv,
		"the ballot form must receive the signing key the registration returned")
	require.Equal(t, "sk-Alice", app.voting.voterPriv,
		"the page must keep the key pair for its copy buttons: it is the only copy")
	require.Contains(t, app.voting.status.text, "Registered Alice")
	require.Empty(t, app.voting.voterName, "a successful registration clears the form")
	require.Equal(t, 0, app.voting.voterChoice, "the new voter must be highlighted")
	require.Empty(t, app.voting.voterSearch, "a search hiding the new voter must be cleared")
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
	app.oracle.tab = 2
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
	app := New(workbenchFakes())
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

	require.True(t, tester.HasText("Window"), "the window must be labelled")
	startLine, endLine := sessionWindowLines(start.Unix(), end.Unix())
	require.True(t, tester.HasText(startLine),
		"the session's start must be on screen")
	require.True(t, tester.HasText(strings.TrimPrefix(endLine, "→ ")),
		"the session's end must be on screen")
	require.NotEqual(t, startLine, endLine, "the window must keep both timestamps distinct")
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

// TestAdvancedFieldsStartCollapsed pins the simplification: fields most first
// uses do not need stay behind a labelled disclosure instead of competing with
// the name/URL/owner they do need. Each still exists and opens on one click.
func TestAdvancedFieldsStartCollapsed(t *testing.T) {
	app := New(workbenchFakes())
	tester := ui.NewTester(app.View, 1280, 5000)

	app.ShowPage("Token")
	tester.Frame()
	require.False(t, tester.HasText("Decimals"), "token decimals start collapsed")
	require.NoError(t, tester.Click("Advanced"))
	tester.Frame()
	require.True(t, tester.HasText("Decimals"))

	require.False(t, tester.HasText("Limit"), "history paging starts collapsed")
	require.NoError(t, tester.Click("Paging"))
	tester.Frame()
	require.True(t, tester.HasText("Limit"))
	require.True(t, tester.HasText("Offset"))

	app.ShowPage("NFT")
	tester.Frame()
	require.False(t, tester.HasText("Description (optional)"), "optional NFT metadata starts collapsed")
	require.NoError(t, tester.Click("Optional details"))
	tester.Frame()
	require.True(t, tester.HasText("Description (optional)"))
	require.True(t, tester.HasText("Image URL (optional)"))

	app.ShowPage("Oracle")
	tester.Frame()
	require.False(t, tester.HasText("Request Headers (optional JSON object)"),
		"oracle transport fields start collapsed")
	require.NoError(t, tester.Click("Advanced"))
	tester.Frame()
	require.True(t, tester.HasText("Request Headers (optional JSON object)"))
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
	require.Equal(t, float64(60), app.oracle.addInterval)

	// And the filled form is addable: no field the service needs is blank.
	require.NotEmpty(t, app.oracle.addURL, "Add Source needs a URL")
}

// TestChoosingAPresetClearsPrivateHeaders keeps one source's credentials from
// travelling into another source. Presets have no headers, so the previous
// form value must be cleared rather than silently attached to the new URL.
func TestChoosingAPresetClearsPrivateHeaders(t *testing.T) {
	fake := &fakeServices{templates: []TemplateRow{{
		ID: "public", Name: "Public feed", URL: "https://example.com/public",
		Method: "GET", Interval: 60,
	}}}
	app := New(fake)
	app.ShowPage("Oracle")
	app.oracle.addHeaders = `{"Authorization":"Bearer private-token"}`

	app.applyTemplate("public")

	require.Empty(t, app.oracle.addHeaders,
		"a preset without headers must not inherit another source's credentials")
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

// TestShortHashIsTheComparedPrefix pins the shared head display rule: twelve
// characters, no ellipsis (the value is copied into comparisons), and a short
// value is untouched.
func TestShortHashIsTheComparedPrefix(t *testing.T) {
	require.Equal(t, "abc123", ShortHash("abc123"))
	full := "0ab2857b160a4c9e2f8d7c6b5a4938271605f4e3d2c1b0a9f8e7d6c5b4a39281"
	require.Equal(t, "0ab2857b160a", ShortHash(full))
	require.True(t, strings.HasPrefix(full, ShortHash(full)),
		"the shown head must be a prefix of the real one")
}

// TestChoosingAHistoryRowFillsTheVerifier pins the path from a draw record to
// the full id the verifier needs: choosing the record or its own Verify button
// loads the draw's real id, which is the only place the summary's ellipsis
// does not cut.
func TestChoosingAHistoryRowFillsTheVerifier(t *testing.T) {
	const full = "54c95915ba8b8f3e"
	fake := &fakeServices{draws: []DrawRow{{ID: full, BlockHeight: 2, Winners: []string{"Dan"}}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 2000)
	app.ShowPage("Lottery")
	tester.Frame()

	require.Empty(t, app.lottery.verifyID)

	require.NoError(t, tester.Click(full), "choose the draw record")
	tester.Frame()
	require.Equal(t, full, app.lottery.verifyID,
		"choosing the record must load the full id into the verifier")
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
	// The collection is a list of records, not a table with a resizable supply
	// column, so the exact supply is drawn on the token's own record and needs
	// no second copy underneath it.
	require.Equal(t, 1, countText(tester.Texts(), "1000"),
		"the exact supply must read on the token's record")
	require.GreaterOrEqual(t, countText(tester.Texts(), "AUR"), 2,
		"the chosen token's id reads on its record and in the subject strip")
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

	require.True(t, tester.HasText(full), "the record must show the full asset id")
	require.NoError(t, tester.Click("First Art"), "choose the asset record")
	tester.Frame()
	require.Equal(t, full, app.nft.id, "Inspect must receive the asset")
	require.Equal(t, full, app.nft.transferID, "Transfer must receive the asset")
	require.Equal(t, full, app.nft.burnID, "Burn must receive the asset")
	require.Equal(t, full, app.nft.historyID, "History must receive the asset")
}

// TestRecordsKeepTheirHeight pins the rhythm that replaced the table's
// clipped rows: a record is as high as its own content, and a long participant
// list wraps inside it rather than being cut off at a row height.
func TestRecordsKeepTheirHeight(t *testing.T) {
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
// action buttons and the read cards. A source record is chosen by name, and
// its full id stays on the record as a field.
func TestChoosingAnOracleRowFillsTheActions(t *testing.T) {
	const full = "20261025233845.347259429-29ee362515b7c1fc06313345ef6be416"
	fake := &fakeServices{}
	fake.sources = []SourceRow{{ID: full, Name: "btc", Enabled: true}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 2400)
	app.ShowPage("Oracle")
	tester.Frame()

	require.True(t, tester.HasText(full), "the record must show the full source id")
	require.NoError(t, tester.Click("btc"), "choose the source record")
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

// TestOraclePresetResetsAndFillsItsInterval pins two edges of the preset
// combobox: a preset without an interval must clear the previous preset's
// value, and adding the source clears the choice along with the fields, since
// a selected preset whose fields are empty cannot be picked again (no change).
func TestOraclePresetResetsAndFillsItsInterval(t *testing.T) {
	fake := &fakeServices{templates: []TemplateRow{
		{ID: "with-interval", Name: "Interval", URL: "https://example.com/a", Method: "GET", Interval: 60},
		{ID: "no-interval", Name: "Default", URL: "https://example.com/b", Method: "GET", Interval: 0},
	}}
	app := New(fake)
	app.ShowPage("Oracle")

	app.applyTemplate("with-interval")
	require.Equal(t, float64(60), app.oracle.addInterval)

	app.applyTemplate("no-interval")
	require.Zero(t, app.oracle.addInterval, "a preset without an interval must clear the previous value")

	app.applyTemplate("with-interval")
	app.addOracleSource()
	require.Empty(t, app.oracle.template, "adding the source must clear the preset choice")
	require.Zero(t, app.oracle.addInterval)
	require.Empty(t, app.oracle.addURL)
}

// TestAddingSourceMakesItTheActionTarget keeps a newly added feed from being
// highlighted but unusable: the shared action field and both read cards must
// move to it, and any readings still open for the previous source must be
// invalidated before the new read.
func TestAddingSourceMakesItTheActionTarget(t *testing.T) {
	boom := errors.New("database is locked")
	fake := &fakeServices{sources: []SourceRow{{ID: "old", Name: "Old source"}}}
	fake.addSource = func(SourceInput) (SourceRow, error) {
		fake.loadErrs.oracleData = boom
		fake.loadErrs.oracleLatest = boom
		return SourceRow{ID: "new", Name: "New source", Type: "json"}, nil
	}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Oracle")
	tester.Frame()

	app.oracle.fetchID, app.oracle.queryID, app.oracle.latestID = "old", "old", "old"
	app.oracle.data = []DataRow{{Value: "old-reading"}}
	app.oracle.dataLoadedSource, app.oracle.dataLoadedLimit = "old", 20
	app.oracle.latest = &DataRow{Value: "old-latest"}
	app.oracle.latestLoadedSource = "old"
	app.oracle.addName, app.oracle.addURL = "New source", "https://example.com/new"

	app.addOracleSource()
	tester.Frame()

	require.Equal(t, "new", app.oracle.fetchID, "the new source must receive the actions")
	require.Equal(t, "new", app.oracle.queryID, "Query must follow the new source")
	require.Equal(t, "new", app.oracle.latestID, "Latest must follow the new source")
	require.Empty(t, app.oracle.data, "the previous source's readings must be dropped")
	require.Nil(t, app.oracle.latest, "the previous source's latest value must be dropped")
	require.Equal(t, "new", fake.lastOracleDataSource)
	require.Equal(t, "new", fake.lastOracleLatestSource)
}

// TestFailedVerifyClearsThePreviousVerdict pins the Verify card's evidence: a
// new lookup that fails must not leave the previous draw's "Verified" panel on
// screen under the new id.
func TestFailedVerifyClearsThePreviousVerdict(t *testing.T) {
	fake := &fakeServices{verifyErr: errors.New("lottery not found")}
	app := New(fake)
	app.lottery.verifyResult = &VerifyResult{ID: "old-draw", Valid: true}
	app.lottery.verifyID = "new-draw"

	app.verifyLottery()

	require.Nil(t, app.lottery.verifyResult, "a failed lookup must clear the old verdict")
	require.Contains(t, app.lottery.verifyErr, "not found")
}

// TestLotteryVerdictHidesWhenTheVerifyInputChanges keeps a successful result
// tied to the exact field value that produced it. Native inputs update the
// state before the next frame; without this check the previous draw's verdict
// remained under a newly typed id until Verify was pressed again.
func TestLotteryVerdictHidesWhenTheVerifyInputChanges(t *testing.T) {
	app := New(&fakeServices{verifyResult: VerifyResult{ID: "draw-a", Valid: true}})
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("Lottery")
	tester.Frame()

	app.lottery.verifyID = "draw-a"
	app.verifyLottery()
	tester.Frame()
	require.True(t, tester.HasText("Verified"))

	app.lottery.verifyID = "draw-b"
	tester.Frame()
	require.False(t, tester.HasText("Verified"),
		"draw-a's verdict must disappear while the field names draw-b")
	require.NotNil(t, app.lottery.verifyResult, "the result is hidden, not discarded")

	app.lottery.verifyID = "draw-a"
	tester.Frame()
	require.True(t, tester.HasText("Verified"),
		"returning to the verified input may show the result that describes it")
}

// TestLotteryVerifyErrorHidesWhenTheInputChanges applies the same subject rule
// to a failed lookup, whose inline error used to outlive the id that caused it.
func TestLotteryVerifyErrorHidesWhenTheInputChanges(t *testing.T) {
	app := New(&fakeServices{verifyErr: errors.New("lottery not found")})
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("Lottery")
	tester.Frame()

	app.lottery.verifyID = "missing-draw"
	app.verifyLottery()
	tester.Frame()
	require.True(t, tester.HasText("lottery not found"))

	app.lottery.verifyID = "another-draw"
	tester.Frame()
	require.False(t, tester.HasText("lottery not found"),
		"the lookup error must disappear while the field names another draw")
}

// TestSelectedSourceShowsItsEndpointAndInterval pins what a source row cannot
// carry: the endpoint it polls and how often. The URL is copyable because the
// readout shortens it.
func TestSelectedSourceShowsItsEndpointAndInterval(t *testing.T) {
	const endpoint = "https://api.example.com/v1/price"
	fake := &fakeServices{sources: []SourceRow{{
		ID: "s1", Name: "btc", URL: endpoint, Type: "price", Enabled: true, Interval: 60,
	}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Oracle")
	tester.Frame()

	require.False(t, tester.HasText("60s"), "no interval before a source is chosen")
	require.NoError(t, tester.Click("s1"))
	tester.Frame()
	require.True(t, tester.HasText("60s"), "the source's polling interval must be shown")
	require.True(t, tester.HasText("URL"))

	require.NoError(t, tester.Click("Copy URL"))
	tester.Frame()
	require.Equal(t, endpoint, tester.Clipboard())
}

// TestOracleSourceActionsRequireASourceID keeps an empty action field from
// sending an empty id to Enable/Disable/Fetch or opening a Delete dialog with
// no target. Once a source is chosen, the same controls become usable.
func TestOracleSourceActionsRequireASourceID(t *testing.T) {
	fake := &fakeServices{sources: []SourceRow{{ID: "s1", Name: "btc", Enabled: true}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Oracle")
	tester.Frame()

	require.True(t, tester.HasText("Choose a source in Sources or enter its ID first."),
		"the action field must say why the buttons are disabled")
	for _, label := range []string{"Enable", "Disable", "Fetch now", "Delete"} {
		require.NoError(t, tester.Click(label))
		tester.Frame()
		require.Zero(t, fake.oracleEnableCalls, "%s must not run without an id", label)
		require.Zero(t, fake.oracleFetchCalls, "%s must not run without an id", label)
		require.Zero(t, fake.oracleDeleteCalls, "%s must not run without an id", label)
		require.False(t, app.oracle.askDelete, "Delete must not open an empty confirmation")
	}

	app.oracle.fetchID = "s1"
	tester.Frame()
	require.NoError(t, tester.Click("Enable"))
	tester.Frame()
	require.Equal(t, 1, fake.oracleEnableCalls)
	require.True(t, fake.lastSourceEnabled)
	require.Equal(t, "s1", fake.lastOracleActionID)

	require.NoError(t, tester.Click("Disable"))
	tester.Frame()
	require.Equal(t, 2, fake.oracleEnableCalls)
	require.False(t, fake.lastSourceEnabled)

	require.NoError(t, tester.Click("Fetch now"))
	tester.Frame()
	require.Equal(t, 1, fake.oracleFetchCalls)
	require.Equal(t, "s1", fake.lastOracleActionID)

	require.NoError(t, tester.Click("Delete"))
	tester.Frame()
	require.True(t, app.oracle.askDelete)
	require.Zero(t, fake.oracleDeleteCalls, "Delete must still wait for confirmation")
}

// TestReadingsStageLoadsTheChosenSource pins the Oracle equivalent: selecting
// a source and opening Readings queries that source without another click.
func TestReadingsStageLoadsTheChosenSource(t *testing.T) {
	fake := &fakeServices{sources: []SourceRow{{ID: "s1", Name: "btc"}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Oracle")
	tester.Frame()

	require.NoError(t, tester.Click("s1"))
	tester.Frame()
	require.Zero(t, fake.oracleDataCalls, "selecting a source alone does not read values")
	require.NoError(t, tester.Click("Readings"))
	tester.Frame()
	require.Positive(t, fake.oracleDataCalls, "Readings must query the selected source")
	require.Positive(t, fake.oracleLatestCalls, "Readings must read the latest value")
}

// TestReadingsDropTheChooseHintOnceASourceIsChosen: the Readings query field
// kept telling the operator to choose a source after one had filled it — the
// same contradiction the session controls had.
func TestReadingsDropTheChooseHintOnceASourceIsChosen(t *testing.T) {
	const hint = "Choose a source in Sources or enter its ID first."
	fake := &fakeServices{sources: []SourceRow{{ID: "s1", Name: "btc"}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Oracle")
	tester.Frame()

	require.NoError(t, tester.Click("Readings"))
	tester.Frame()
	require.True(t, tester.HasText(hint), "with nothing chosen the hint names the first step")

	require.NoError(t, tester.Click("Sources"))
	tester.Frame()
	require.NoError(t, tester.Click("s1"))
	tester.Frame()
	require.False(t, tester.HasText(hint),
		"the actions card must not ask once a source is chosen")
	require.NoError(t, tester.Click("Readings"))
	tester.Frame()
	require.False(t, tester.HasText(hint), "neither card may contradict the filled field")
	require.Equal(t, "s1", app.oracle.queryID, "the field still holds the chosen source")
}

// TestOracleQueryLimitDefaultsToTheSharedPageSize: the Query card's untouched
// limit is the same 10 that REST's /oracle/query and the CLI's `oracle data`
// default to, so the three surfaces ask the store for the same page.
func TestOracleQueryLimitDefaultsToTheSharedPageSize(t *testing.T) {
	app := New(&fakeServices{})
	require.Equal(t, 10, app.oracleQueryLimit())
	app.oracle.queryLim = 0
	require.Equal(t, 10, app.oracleQueryLimit(), "a cleared field falls back to the same page size")
	app.oracle.queryLim = 5000
	require.Equal(t, maxListLimit, app.oracleQueryLimit(), "the cap still holds")
}

// TestSessionTimeErrorsShowBelowTheField covers the two session time controls.
// They used to report a message twice — through attachFieldError on the wrapping
// Field and again through Error on the inner TimeInput — and only the first of
// those draws anything: the inner element carries no field of its own, so that
// call was a no-op that made the code read as if it were doing something.
//
// The assertion is on the rendered text, which is what a reader sees.
func TestSessionTimeErrorsShowBelowTheField(t *testing.T) {
	fake := &fakeServices{voterRows: []VoterRow{{Name: "Ada", PublicKey: "pk-1"}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Voting")
	tester.Frame()

	const startMsg = "start must not be in the past"
	const endMsg = "end must be after start"
	app.voting.sessionStartErr, app.voting.sessionEndErr = startMsg, endMsg
	// The first frame after typing clears a field's message, so the errors are
	// set after the page has settled and read on the next one.
	tester.Frame()

	texts := strings.Join(tester.Texts(), "\n")
	require.Contains(t, texts, startMsg, "the start message must be on screen")
	require.Contains(t, texts, endMsg, "the end message must be on screen")
	require.Equal(t, 1, strings.Count(texts, startMsg), "the start message must appear once")
	require.Equal(t, 1, strings.Count(texts, endMsg), "the end message must appear once")
}

// TestEveryFieldErrorReachesTheScreen counts rendered copies of each form's
// error message. A field error that draws nothing is invisible: the write is
// refused, the button stays disabled-looking, and the reader has no sentence
// telling them which input to fix. The count is asserted as exactly one so a
// second call reporting the same message also fails.
//
// The setup is per case because a page's cards depend on the store: Setup shows
// the registration forms over an empty roster and the session card only once a
// voter exists, and Create Token keeps its decimals input behind a collapsed
// Advanced section.
func TestEveryFieldErrorReachesTheScreen(t *testing.T) {
	cases := []struct {
		name   string
		page   string
		msg    string
		seed   *fakeServices
		before func(*App)
		set    func(*App, string)
	}{
		{
			name: "lottery participants", page: "Lottery", msg: "PARTICIPANTS_BOOM",
			seed: &fakeServices{},
			set:  func(a *App, m string) { a.lottery.participantsErr = m },
		},
		{
			name: "lottery seed", page: "Lottery", msg: "SEED_BOOM",
			seed: &fakeServices{},
			set:  func(a *App, m string) { a.lottery.seedErr = m },
		},
		{
			name: "lottery count", page: "Lottery", msg: "COUNT_BOOM",
			seed: &fakeServices{},
			set:  func(a *App, m string) { a.lottery.countErr = m },
		},
		{
			name: "voter name", page: "Voting", msg: "VOTER_BOOM",
			seed: &fakeServices{voterRows: []VoterRow{{Name: "Ada", PublicKey: "pk-1"}}},
			set:  func(a *App, m string) { a.voting.voterNameErr = m },
		},
		{
			name: "candidate name", page: "Voting", msg: "CAND_BOOM",
			seed: &fakeServices{voterRows: []VoterRow{{Name: "Ada", PublicKey: "pk-1"}}},
			set:  func(a *App, m string) { a.voting.candNameErr = m },
		},
		{
			name: "session start", page: "Voting", msg: "START_BOOM",
			seed: &fakeServices{voterRows: []VoterRow{{Name: "Ada", PublicKey: "pk-1"}}},
			set:  func(a *App, m string) { a.voting.sessionStartErr = m },
		},
		{
			name: "session end", page: "Voting", msg: "END_BOOM",
			seed: &fakeServices{voterRows: []VoterRow{{Name: "Ada", PublicKey: "pk-1"}}},
			set:  func(a *App, m string) { a.voting.sessionEndErr = m },
		},
		{
			name: "token decimals", page: "Token", msg: "DECIMALS_BOOM",
			seed:   &fakeServices{tokenRows: []TokenRow{{ID: "GLD", Name: "Gold"}}},
			before: func(a *App) { a.token.createAdvanced = true },
			set:    func(a *App, m string) { a.token.decimalsErr = m },
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app := New(tc.seed)
			tester := ui.NewTester(app.View, 1400, 8000)
			app.ShowPage(tc.page)
			tester.Frame()
			if tc.before != nil {
				tc.before(app)
				tester.Frame()
			}
			tc.set(app, tc.msg)
			tester.Frame()

			texts := strings.Join(tester.Texts(), "\n")
			require.Equalf(t, 1, strings.Count(texts, tc.msg),
				"the message must appear exactly once in what the reader sees")
		})
	}
}

// TestTheSessionCardDropsItsChooseHintOnceAnIDIsTyped covers the other half of
// the hint rule on the Voting ballot card. The existing test types an id and
// checks that the buttons start working; it never checks that the sentence telling
// the operator to choose a session went away, so the card could keep contradicting
// a filled field while every button worked.
func TestTheSessionCardDropsItsChooseHintOnceAnIDIsTyped(t *testing.T) {
	const hint = "Choose a stored session below, or paste its ID."
	fake := &fakeServices{
		voterRows:  []VoterRow{{PublicKey: "voter-pk-1", Name: "Ada"}},
		sessions:   []SessionRow{{ID: "s1", Title: "Budget"}},
		candidates: []CandidateRow{{ID: "cand-1", Name: "Ada"}},
	}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	app.voting.tab = 1
	tester.Frame()
	require.True(t, tester.HasText(hint), "with no id the hint names the first step")

	app.voting.controlID = "s1"
	tester.Frame()
	require.False(t, tester.HasText(hint),
		"a filled id must not sit beside the sentence telling the operator to choose one")

	// Clearing it brings the first step back: the hint follows the field, not the
	// fact that this card was visited once.
	app.voting.controlID = ""
	tester.Frame()
	require.True(t, tester.HasText(hint), "a cleared field needs the first step again")
}

// failingVoterRegistration refuses every registration while still returning a key
// pair in the value, which is what a store that wrote the row and then failed its
// bookkeeping could hand back. The window must ignore the row entirely.
type failingVoterRegistration struct {
	*fakeServices
}

func (f *failingVoterRegistration) RegisterVoter(name string) (VoterRow, error) {
	return VoterRow{PublicKey: "pk-" + name, Name: name, PrivateKey: "sk-" + name},
		errors.New("voter name already registered")
}

// TestAFailedRegistrationLeaksNoKey covers the path a duplicate name takes. The
// success path is tested — the private key is the one thing only this response
// carries — but nothing checked that a refused registration leaves no key behind.
// The value can carry one even on failure (a store may write the row and then fail
// its bookkeeping), so the guard is the error, not the empty struct.
func TestAFailedRegistrationLeaksNoKey(t *testing.T) {
	app := New(&failingVoterRegistration{fakeServices: &fakeServices{}})
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("Voting")
	tester.Frame()

	app.voting.voterName = "Alice"
	app.registerVoter()
	// The status line is drawn by the view, so the frame has to run before the
	// sentence can be found on screen.
	tester.Frame()

	require.True(t, app.voting.status.isErr)
	require.Contains(t, app.voting.status.text, "already registered")
	require.Empty(t, app.voting.voterPriv,
		"a refused registration must not fill the ballot form's key")
	require.NotEqual(t, "sk-Alice", app.voting.votePriv,
		"and the refused key must never reach the ballot's signing field")
	require.Empty(t, app.voting.voterPub, "nor its public half")
	require.True(t, tester.HasText(app.voting.status.text))
}

// TestTheSourcePathFieldSaysWhatAShapelessPathCannot covers the one field in Add Source
// whose syntax is not guessable. extractByPath walks object keys separated by dots and
// supports nothing else — no array index, no bracket syntax — so "a.b[0].c" never
// resolves and the fetch fails with an invalid-source verdict that names no syntax.
//
// The CLI has said so for a while ("dotted, e.g. bitcoin.usd"); the window's field said
// only "optional JSON field", and it is behind a disclosure that opens by itself only
// when a preset filled it.
func TestTheSourcePathFieldSaysWhatAShapelessPathCannot(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1400, 4000)
	app.ShowPage("Oracle")
	app.oracle.addAdvanced = true
	tester.Frame()

	const hint = "Dotted object keys only, e.g. bitcoin.usd; no array index."
	require.True(t, tester.HasText(hint),
		"the path field must state its syntax where the operator types it")

	// The example is a real one: the resolver walks it.
	require.True(t, tester.HasText(tr("Path (optional JSON field)")),
		"and the field itself must still be drawn under that hint")
}

// TestAPresetIsAFillNotAMode covers what the preset box does after the reader edits a
// field it filled. The box keeps its name — the toolkit's value is the page's own
// string — while Add Source submits the fields, not the preset, so an edited form and a
// named preset disagree and nothing said which would be saved.
//
// The sentence has to be on screen, not only in the docs: the question arises at the
// moment of filling the form, which is exactly when the docs are not open.
func TestAPresetIsAFillNotAMode(t *testing.T) {
	app := New(&fakeServices{templates: []TemplateRow{{
		ID: "bitcoin", Name: "Bitcoin", URL: "https://api.example.com/btc",
		Method: "GET", Path: "bitcoin.usd", Interval: 60,
	}}})
	tester := ui.NewTester(app.View, 1400, 4000)
	app.ShowPage("Oracle")
	tester.Frame()

	// The box's change callback is what fills the form, so the choice and the fill
	// are made together, as the frame does when the box changes.
	app.oracle.template = "bitcoin — Bitcoin"
	app.applyTemplate("bitcoin")
	tester.Frame()
	require.Equal(t, "https://api.example.com/btc", app.oracle.addURL,
		"choosing a preset fills the fields")
	require.True(t, app.oracle.addAdvanced, "and opens the disclosure that hides them")

	// The reader replaces the endpoint. The form is now what gets saved.
	app.oracle.addURL = "https://internal.example.com/price"
	tester.Frame()
	require.True(t, tester.HasText(tr("A preset fills the fields once; edit them freely afterwards.")),
		"the form must say the preset does not keep overwriting what was typed")

	// What is submitted is the edited form, never the preset.
	var submitted SourceInput
	fake := &fakeServices{addSource: func(in SourceInput) (SourceRow, error) {
		submitted = in
		return SourceRow{ID: "src", Name: in.Name, URL: in.URL}, nil
	}}
	app.services = fake
	app.oracle.addName, app.oracle.addType = "priced", "bitcoin"
	app.addOracleSource()
	require.Equal(t, "https://internal.example.com/price", submitted.URL,
		"the edited URL is what reaches the store")
}

// TestTheHealthCardSaysTheCountsStartAtLaunch covers the boundary the numbers alone
// cannot show. The counters live in the scheduler's memory, not the database: they
// begin at zero when the process does. A row reading "Attempts 0, Successes 0,
// Failures 0" therefore means "nothing has run since you launched this", not "this feed
// has never had a problem", and the card said neither.
func TestTheHealthCardSaysTheCountsStartAtLaunch(t *testing.T) {
	app := New(&fakeServices{healthRows: []HealthRow{
		{SourceID: "s1", Attempts: 0, Successes: 0, Failures: 0},
		{SourceID: "s2", Attempts: 9, Successes: 8, Failures: 1, LastError: "503 Service Unavailable"},
	}})
	tester := ui.NewTester(app.View, 1400, 4000)
	app.ShowPage("Oracle")
	app.oracle.tab = 1
	tester.Frame()

	require.True(t,
		tester.HasText(tr("Background poll attempts, successes and failures per source since this launch, with the last error kept verbatim.")),
		"the card must scope its counters to this launch")

	// A quiet feed and a busy one must not look the same, and both stay readable.
	require.True(t, tester.HasText("503 Service Unavailable"),
		"the last error is kept verbatim")
	require.True(t, tester.HasText("Feed Health"))
}

// TestAZeroAllowanceIsShownNotHidden covers the one readout whose empty-looking value is
// real data. The card draws the row when the stored answer is not the empty string, and
// "0" is not empty: a spender who was granted nothing, or who spent everything, has an
// answer of zero, and hiding it would leave the reader unable to tell that from "you have
// not asked yet".
//
// The condition is asserted on the source rather than on the frame: the page draws the
// word "Allowance" as the card's title too, so a text search cannot separate "the card is
// there" from "the answer is there" — the two differ by one row, not by a word.
func TestAZeroAllowanceIsShownNotHidden(t *testing.T) {
	source := readFile(t, "token_read.go")

	require.Contains(t, source, `if a.token.allowance != "" && matches {`,
		"the row must be drawn whenever a value was read for the account on screen")
	require.NotContains(t, source, `a.token.allowance != "0"`,
		`a zero allowance is an answer, not an absence: nothing may special-case it away`)

	// And the empty string keeps meaning "not asked yet", which is the case the guard is
	// for — a cleared account must not leave the previous answer under a new key.
	require.Contains(t, source, "func (s tokenState) allowanceMatches(",
		"the account the answer belongs to is what the guard pairs with the value")
}

// TestAnOverlongTokenFieldIsRejectedAtItsControl covers where the token form's
// length refusals belong. The store bounds the name and the symbol and its message
// names no field, so both used to surface in the status line above a form whose
// controls sit on two different rows.
//
// The sentences are the domain's own: a window-side copy could describe a limit
// the store does not apply, and the CLI would then disagree with the window.
func TestAnOverlongTokenFieldIsRejectedAtItsControl(t *testing.T) {
	for _, tc := range []struct {
		field string
		name  string
		tick  string
		check func(*App) string
	}{
		{"name", strings.Repeat("N", token.MaxTokenNameLength+1), "GLD", func(a *App) string { return a.token.nameErr }},
		{"symbol", "Gold", strings.Repeat("S", token.MaxTokenSymbolLength+1), func(a *App) string { return a.token.symbolErr }},
	} {
		t.Run(tc.field, func(t *testing.T) {
			fake := workbenchFakes()
			app := New(fake)
			tester := ui.NewTester(app.View, 1280, 2600)
			app.ShowPage("Token")
			tester.Frame()
			before := fake.createTokenCalls

			app.token.name, app.token.symbol = tc.name, tc.tick
			app.token.supply, app.token.owner = "100", "pk"
			app.createToken()
			tester.Frame()

			require.Equal(t, before, fake.createTokenCalls,
				"a %s the store would refuse must not reach it", tc.field)
			require.NotEmpty(t, tc.check(app), "the refusal belongs on the field it applies to")
			require.True(t, tester.HasText(tc.check(app)))
		})
	}
}

// TestTokenFieldsAtTheirLimitsAreAccepted keeps the check from being too eager on
// either side. The bounds are inclusive, so a value exactly at a limit is valid
// and must reach storage.
func TestTokenFieldsAtTheirLimitsAreAccepted(t *testing.T) {
	fake := workbenchFakes()
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 2600)
	app.ShowPage("Token")
	tester.Frame()
	before := fake.createTokenCalls

	app.token.name = strings.Repeat("N", token.MaxTokenNameLength)
	app.token.symbol = strings.Repeat("S", token.MaxTokenSymbolLength)
	app.token.supply, app.token.owner = "100", "pk"
	app.createToken()
	tester.Frame()

	require.Equal(t, before+1, fake.createTokenCalls, "both bounds are inclusive")
	require.Empty(t, app.token.nameErr)
	require.Empty(t, app.token.symbolErr)
}

// TestTheParticipantRefusalReadsAsAFieldMessage covers the sentence under the
// participants field. The domain wraps a per-name failure as "invalid participant:
// <reason>" because its other caller prints a whole CLI sentence; under a control
// already labelled "Participants (comma-separated)" that prefix repeats the label
// and buries the reason.
func TestTheParticipantRefusalReadsAsAFieldMessage(t *testing.T) {
	for _, tc := range []struct{ name, input, want string }{
		{"invalid character", "Ada,Bob<script>", "participant name contains invalid characters"},
		{"too long", "Ada," + strings.Repeat("N", lottery.MaxParticipantNameLength+1), "participant name too long"},
		{"duplicate", "Ada,Bob,Ada", "duplicate participant"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := New(workbenchFakes())
			tester := ui.NewTester(app.View, 1280, 2600)
			app.ShowPage("Lottery")
			tester.Frame()

			app.lottery.participants, app.lottery.seed, app.lottery.count = tc.input, "seed-abc", 1
			app.createLottery()
			tester.Frame()

			require.Equal(t, tc.want, app.lottery.participantsErr,
				"the reason must stand on its own under the field")
			require.NotContains(t, app.lottery.participantsErr, "invalid participant:")
			require.True(t, tester.HasText(tc.want))
		})
	}
}

// TestBlankParticipantEntriesAreIgnored keeps the field forgiving about the comma
// list's shape. A trailing comma or a doubled one is a typing accident, not a
// participant named "": dropping the empty entries is what lets an operator paste
// a list and move on.
func TestBlankParticipantEntriesAreIgnored(t *testing.T) {
	app := New(workbenchFakes())
	tester := ui.NewTester(app.View, 1280, 2600)
	app.ShowPage("Lottery")
	tester.Frame()

	app.lottery.participants, app.lottery.seed, app.lottery.count = "Ada, , Bob,", "seed-abc", 1
	app.createLottery()
	tester.Frame()

	require.Empty(t, app.lottery.participantsErr,
		"blank entries between commas must not become participants")
	require.Contains(t, app.lottery.status.text, "recorded")
}

// TestTheCreateButtonReachesItsHandler clicks the control a reader clicks.
//
// Every other create test calls app.createToken() directly, which proves the
// handler and skips the button entirely: a wrong allSet argument or a broken
// wiring between the control and the handler would pass all of them. The click
// is placed by coordinate because Tester.Find returns the first element showing
// the text, which here is the whole card, and clicking a card's centre lands on
// a form field.
func TestTheCreateButtonReachesItsHandler(t *testing.T) {
	app := New(&fakeServices{})
	app.SetActions(Actions{Version: "0.0.1", SetPageTitle: func(string) {}, SetBadgeCount: func(int) {}})
	tester := ui.NewTester(app.View, 1280, 1400)
	app.ShowPage("Token")
	app.token.name, app.token.symbol = "Gold", "GLD"
	app.token.supply, app.token.owner = "100.5", "pk"
	tester.Frame()

	// A decimal supply is refused under the field, which is the visible sign
	// that the handler ran at all.
	clickPrimaryButton(t, tester, "Create Token",
		func() bool { return app.token.supplyErr != "" },
		func() { app.token.supplyErr = "" })
	require.Contains(t, app.token.supplyErr, "whole number",
		"and the refusal is the one the handler produces")
}

// TestEveryPrimaryButtonReachesItsHandler covers the controls a reader clicks.
// Their tests call the handlers directly, which proves the handler and skips the
// control: a wrong allSet argument or a broken wiring between button and handler
// passes every one of them. Each case fills the form, clicks the button by
// coordinate, and states the change only its handler makes.
func TestEveryPrimaryButtonReachesItsHandler(t *testing.T) {
	cases := []struct {
		name   string
		page   string
		label  string
		fill   func(*App)
		reset  func(*App)
		landed func(*App) bool
	}{
		{
			name: "create draw", page: "Lottery", label: "Create Draw",
			fill:   func(a *App) { a.lottery.participants, a.lottery.seed = "A,B", "seed-value" },
			reset:  func(a *App) { a.lottery.status = statusLine{} },
			landed: func(a *App) bool { return a.lottery.status.text != "" },
		},
		{
			name: "register voter", page: "Voting", label: "Register Voter",
			fill:   func(a *App) { a.voting.voterName = "Ada" },
			reset:  func(a *App) { a.voting.voterNameErr = "" },
			landed: func(a *App) bool { return a.voting.voterNameErr == "" && a.voting.voterName == "" },
		},
		{
			name: "register candidate", page: "Voting", label: "Register Candidate",
			fill:   func(a *App) { a.voting.candName = "Alice" },
			reset:  func(a *App) { a.voting.candNameErr = "" },
			landed: func(a *App) bool { return a.voting.candNameErr == "" && a.voting.candName == "" },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := New(workbenchFakes())
			app.SetActions(Actions{Version: "0.0.1", SetPageTitle: func(string) {}, SetBadgeCount: func(int) {}})
			tester := ui.NewTester(app.View, 1400, 2000)
			app.ShowPage(tc.page)
			tc.fill(app)
			tester.Frame()

			clickPrimaryButton(t, tester, tc.label,
				func() bool { return tc.landed(app) },
				func() { tc.reset(app) })
		})
	}
}

// TestTheSessionAndMintButtonsReachTheirHandlers covers the two primary buttons a
// test can reach without a key pair. Both were missed by the earlier sweep: the
// session button was clicked all along, but the test watched the wrong field for
// the result, and the mint card's subtitle is not a text node that Find answers
// with, so the field above the button is used as the anchor instead.
func TestTheSessionAndMintButtonsReachTheirHandlers(t *testing.T) {
	t.Run("create session", func(t *testing.T) {
		app := New(workbenchFakes())
		app.SetActions(Actions{Version: "0.0.1", SetPageTitle: func(string) {}, SetBadgeCount: func(int) {}})
		tester := ui.NewTester(app.View, 1400, 2400)
		app.ShowPage("Voting")
		app.voting.sessionTitle = "Election"
		tester.Frame()

		// An empty roster is refused, which is this handler's own first step.
		clickPrimaryButton(t, tester, "Create Session",
			func() bool { return app.voting.sessionResult.text != "" },
			func() { app.voting.sessionResult = statusLine{} })
		require.Contains(t, app.voting.sessionResult.text, "candidate",
			"and the refusal is the one the handler produces")
	})

	t.Run("mint", func(t *testing.T) {
		app := New(workbenchFakes())
		app.SetActions(Actions{Version: "0.0.1", SetPageTitle: func(string) {}, SetBadgeCount: func(int) {}})
		tester := ui.NewTester(app.View, 1400, 2400)
		app.ShowPage("Token")
		tester.Frame()
		// Selecting a token fills the recipient and keeps the fields on screen;
		// a subject change resets the write forms by design.
		app.token.tokenID = "GLD"
		app.rememberTokenSubjectOwner("GLD", "owner-pk")
		tester.Frame()
		app.token.mintAmount, app.token.mintPriv = "100.5", "priv"
		tester.Frame()

		// The card's subtitle is not a text node Find can answer with, so the
		// search starts from the field above the button.
		field, ok := tester.Find("To (base64 public key)")
		require.True(t, ok)
		require.Equal(t, "", app.token.amountErrs["mint"])

		landed := false
		for y := field.Y + 40; y < field.Y+220 && !landed; y += 6 {
			app.clearAmountErr("mint")
			tester.ClickAt(field.X+40, y)
			tester.Frame()
			landed = app.token.amountErrs["mint"] != ""
		}
		require.True(t, landed, "clicking Mint must reach mintToken()")
		require.Contains(t, app.token.amountErrs["mint"], "whole number")
	})
}

package nativeui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// tokenState is the Token page's state.
type tokenState struct {
	// Create form.
	name     string
	symbol   string
	supply   string
	decimals string
	owner    string

	// Read forms.
	tokenID string
	account string

	info    *TokenRow
	balance string

	// Write forms.
	mintTo         string
	mintAmount     string
	mintPriv       string
	xFrom          string
	xTo            string
	xAmount        string
	xPriv          string
	approveOwner   string
	approveTo      string
	approveAmt     string
	approvePriv    string
	fromOwner      string
	fromTo         string
	fromAmount     string
	fromPriv       string
	burnFrom       string
	burnAmount     string
	burnPriv       string
	allowanceOwner string
	allowanceTo    string
	historyLimit   string

	allowance string
	history   []TransferRow

	// tokens is every stored token. It is what lets an operator pick one
	// instead of already knowing its id, which nothing in the window showed.
	tokens []TokenRow
	// chosen is the row chosen in that list, -1 for none.
	chosen int

	loadErr error

	// askBurn drives the irreversible-burn confirmation. The burn only runs
	// when the operator picks the destructive button, mirroring the CLI's
	// --confirm gate.
	askBurn bool

	status statusLine

	historyTable ui.ListState
	list         ui.ListState
}

func (a *App) loadToken() {
	if a.services == nil {
		return
	}
	// The list is the page's entry point: without it a token created in an
	// earlier session could not be reached at all.
	tokens, err := a.services.ListTokens()
	a.token.loadErr = err
	if err == nil {
		a.token.tokens = tokens
	}
}

// tokenView lays the page out in two columns: the token you are working with
// on the left (create, inspect, history), and the operations that change it on
// the right (mint, transfer, approve, transfer-from, burn, allowance). Ten
// full-width forms stacked in one column buried every write below the fold.
func (a *App) tokenView(c *ui.Context) {
	ui.Scroll(c).Fill().Children(func() {
		twoColumns(c, func() {
			a.tokenList(c)
			a.tokenCreate(c)
			a.tokenRead(c)
			a.tokenHistory(c)
		}, func() {
			a.tokenWrites(c)
		})
		a.renderBurnDialog(c)
	})
}

// tokenList draws the stored tokens and lets the operator choose one: a
// choice fills the id every other card acts on, so a token created in an
// earlier session is reachable without remembering its id.
func (a *App) tokenList(c *ui.Context) {
	formCard(c, "Tokens", "Every stored token. Choose one to make it the subject of the cards here.", func() {
		if a.token.loadErr != nil && !loadWarning(c, "Failed to load tokens", a.token.loadErr, len(a.token.tokens) > 0) {
			return
		}
		if len(a.token.tokens) == 0 {
			emptyNote(c, "No tokens yet. Create one below.")
			return
		}
		rows := a.token.tokens
		a.token.list.Selected = &a.token.chosen
		table := ui.Table(c, &a.token.list, []ui.TableColumn{
			{Title: "ID", Width: 150},
			{Title: "Name", MinWidth: 120},
			{Title: "Symbol", Width: 90},
			{Title: "Supply", Width: 130, Align: ui.End},
		}, len(rows), func(row, col int) {
			t := rows[row]
			switch col {
			case 0:
				ui.Text(c, shortID(t.ID)).SingleLine()
			case 1:
				ui.Text(c, t.Name).SingleLine()
			case 2:
				ui.Text(c, t.Symbol)
			case 3:
				ui.Text(c, t.TotalSupply)
			}
		})
		if table.Changed() && a.token.chosen >= 0 && a.token.chosen < len(rows) {
			a.token.tokenID = rows[a.token.chosen].ID
		}
	})
}

func (a *App) tokenCreate(c *ui.Context) {
	formCard(c, "Create Token", "Define supply and attach the owner key that will authorize future writes.", func() {
		field(c, "Name", &a.token.name)
		field(c, "Symbol", &a.token.symbol)
		field(c, "Total Supply", &a.token.supply)
		fieldHint(c, "Decimals", "0 for a whole-unit token; 8 matches most asset conventions.", &a.token.decimals)
		fieldHint(c, "Owner (base64 public key)", "Required: the owner signs every later mint, transfer and burn.", &a.token.owner)
		ui.Row(c).Children(func() {
			if primaryButton(c, "Create Token").Clicked() {
				a.createToken()
			}
		})
	})
}

func (a *App) createToken() {
	a.token.status = statusLine{}
	decimals := 8
	if s := strings.TrimSpace(a.token.decimals); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 || n > 127 {
			a.token.status = statusLine{text: "Decimals must be a whole number between 0 and 127.", isErr: true}
			return
		}
		decimals = n
	}
	t, err := a.services.CreateToken(
		strings.TrimSpace(a.token.name),
		strings.TrimSpace(a.token.symbol),
		strings.TrimSpace(a.token.supply),
		strings.TrimSpace(a.token.owner),
		decimals,
	)
	if err != nil {
		a.token.status = errStatus(err)
		return
	}
	a.token.status = okStatus("Created %s (%s) with supply %s. Owner: %s", t.Name, t.Symbol, t.TotalSupply, t.Owner)
	a.token.tokenID = t.ID
	// The list is the page's entry point, so a new token joins it at once and
	// the row it lands on is the one chosen.
	a.loadToken()
	a.token.name, a.token.symbol, a.token.supply = "", "", ""
}

func (a *App) tokenRead(c *ui.Context) {
	formCard(c, "Inspect", "Read a token's metadata and an account's balance.", func() {
		a.tokenReadBody(c)
	})
}

func (a *App) tokenReadBody(c *ui.Context) {
	field(c, "Token ID", &a.token.tokenID)
	ui.Row(c).Children(func() {
		if secondaryButton(c, "Get Token Info").Clicked() {
			t, err := a.services.TokenInfo(strings.TrimSpace(a.token.tokenID))
			if err != nil {
				a.token.status = errStatus(err)
				a.token.info = nil
			} else {
				a.token.info = &t
				a.token.status = statusLine{}
			}
		}
	})
	if info := a.token.info; info != nil {
		readout(c, []readoutRow{
			{"Name", info.Name + " (" + info.Symbol + ")"},
			{"Supply", info.TotalSupply},
			{"Decimals", strconv.Itoa(info.Decimals)},
			{"Owner", info.Owner},
		})
	}

	field(c, "Account (base64 public key)", &a.token.account)
	ui.Row(c).Children(func() {
		if secondaryButton(c, "Get Balance").Clicked() {
			bal, err := a.services.TokenBalance(strings.TrimSpace(a.token.tokenID), strings.TrimSpace(a.token.account))
			if err != nil {
				a.token.status = errStatus(err)
				a.token.balance = ""
			} else {
				a.token.balance = bal
				a.token.status = statusLine{}
			}
		}
	})
	if a.token.balance != "" {
		readout(c, []readoutRow{{"Balance", a.token.balance}})
	}
}

// readoutRow is one label/value pair in a readout.
type readoutRow struct{ label, value string }

// readout draws the result of a read as label/value rows on a recessed panel,
// so an answer is visibly an answer rather than another line of prose that
// looks like the hint above it.
func readout(c *ui.Context, rows []readoutRow) {
	ui.Column(c).FillWidth().Padding(10).Gap(4).Radius(8).
		Background(ui.Hex(night)).Children(func() {
		for _, r := range rows {
			ui.Row(c).Gap(8).Children(func() {
				ui.Text(c, r.label).FontSize(11).Width(72).TextColor(ui.Hex(inkMuted))
				ui.Text(c, r.value).FontSize(12).TextColor(ui.Hex(ink)).SingleLine()
			})
		}
	})
}

// tokenSubject names the token every write on this page acts on.
//
// The cards take no token id of their own: they all use a.token.tokenID, which
// Create and Inspect set. Without saying so, the right-hand column gave no
// sign of which token a Mint would hit — the one thing an operator must be
// sure of before a write.
func tokenSubject(c *ui.Context, id string) {
	if id == "" {
		ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Padding(10).Radius(8).
			Background(ui.Hex(warningTint)).BorderWidth(1).BorderColor(ui.Hex(auroraAmber)).
			Children(func() {
				badge(c, "No token selected", auroraAmber, night)
				ui.Text(c, "Create or inspect one first.").FontSize(12).TextColor(ui.Hex(ink))
			})
		return
	}
	ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Padding(10).Radius(8).
		Background(ui.Hex(night)).Children(func() {
		ui.Text(c, "Acting on").FontSize(11).TextColor(ui.Hex(inkMuted))
		ui.Text(c, id).FontSize(13).Bold().TextColor(ui.Hex(aurora)).SingleLine()
	})
}

func (a *App) tokenWrites(c *ui.Context) {
	tokenSubject(c, a.token.tokenID)
	formCard(c, "Mint", "Only the token owner can mint, and only for a mintable token.", func() {
		field(c, "To (base64 public key)", &a.token.mintTo)
		field(c, "Amount", &a.token.mintAmount)
		field(c, "Owner Private Key", &a.token.mintPriv)
		ui.Row(c).Children(func() {
			if primaryButton(c, "Mint").Clicked() {
				a.mintToken()
			}
		})
	})

	formCard(c, "Transfer", "Move tokens from one account to another.", func() {
		field(c, "From (base64 public key)", &a.token.xFrom)
		field(c, "To (base64 public key)", &a.token.xTo)
		field(c, "Amount", &a.token.xAmount)
		field(c, "From Private Key", &a.token.xPriv)
		ui.Row(c).Children(func() {
			if primaryButton(c, "Transfer").Clicked() {
				a.transferToken()
			}
		})
	})

	formCard(c, "Approve", "Let a spender move up to an amount on the owner's behalf.", func() {
		field(c, "Owner (base64 public key)", &a.token.approveOwner)
		field(c, "Spender (base64 public key)", &a.token.approveTo)
		field(c, "Amount", &a.token.approveAmt)
		field(c, "Owner Private Key", &a.token.approvePriv)
		ui.Row(c).Children(func() {
			if primaryButton(c, "Approve").Clicked() {
				a.approveToken()
			}
		})
	})

	formCard(c, "Transfer From", "Spend an allowance the owner granted.", func() {
		field(c, "Owner (base64 public key)", &a.token.fromOwner)
		field(c, "To (base64 public key)", &a.token.fromTo)
		field(c, "Amount", &a.token.fromAmount)
		field(c, "Spender Private Key", &a.token.fromPriv)
		ui.Row(c).Children(func() {
			if primaryButton(c, "Transfer From").Clicked() {
				a.transferFromToken()
			}
		})
	})

	formCard(c, "Burn", "Destroy tokens from an account. This cannot be undone.", func() {
		field(c, "From (base64 public key)", &a.token.burnFrom)
		field(c, "Amount", &a.token.burnAmount)
		field(c, "Owner Private Key", &a.token.burnPriv)
		ui.Row(c).Children(func() {
			// A destructive action carries the danger style, and the
			// confirmation still stands between it and the write.
			if dangerButton(c, "Burn…").Clicked() {
				a.token.askBurn = true
			}
		})
	})

	formCard(c, "Allowance", "Read how much a spender may still move.", func() {
		field(c, "Owner (base64 public key)", &a.token.allowanceOwner)
		field(c, "Spender (base64 public key)", &a.token.allowanceTo)
		ui.Row(c).Children(func() {
			if secondaryButton(c, "Get Allowance").Clicked() {
				a.loadAllowance()
			}
		})
		if a.token.allowance != "" {
			readout(c, []readoutRow{{"Allowance", a.token.allowance}})
		}
	})
}

// The write handlers below read only the inputs of the section that invoked
// them. Sharing one address field between sections (as the page once did for
// Approve and Allowance) signs for an account the operator typed somewhere
// else and cannot see from the form they used.

func (a *App) mintToken() {
	err := a.services.MintToken(
		strings.TrimSpace(a.token.tokenID), strings.TrimSpace(a.token.mintTo),
		strings.TrimSpace(a.token.mintAmount), strings.TrimSpace(a.token.mintPriv))
	a.token.status = wrapErr(err)
	if err == nil {
		a.refreshToken()
	}
}

func (a *App) transferToken() {
	err := a.services.TransferToken(
		strings.TrimSpace(a.token.tokenID), strings.TrimSpace(a.token.xFrom),
		strings.TrimSpace(a.token.xTo), strings.TrimSpace(a.token.xAmount),
		strings.TrimSpace(a.token.xPriv))
	a.token.status = wrapErr(err)
	if err == nil {
		a.refreshToken()
	}
}

func (a *App) approveToken() {
	err := a.services.ApproveToken(
		strings.TrimSpace(a.token.tokenID), strings.TrimSpace(a.token.approveOwner),
		strings.TrimSpace(a.token.approveTo), strings.TrimSpace(a.token.approveAmt),
		strings.TrimSpace(a.token.approvePriv))
	a.token.status = wrapErr(err)
	if err == nil {
		a.refreshToken()
	}
}

func (a *App) transferFromToken() {
	err := a.services.TransferFromToken(
		strings.TrimSpace(a.token.tokenID), strings.TrimSpace(a.token.fromOwner),
		strings.TrimSpace(a.token.fromTo), strings.TrimSpace(a.token.fromAmount),
		strings.TrimSpace(a.token.fromPriv))
	a.token.status = wrapErr(err)
	if err == nil {
		a.refreshToken()
	}
}

// refreshToken re-reads what a write just changed: the collection (whose rows
// show total supply) and, when it is open, the Inspect readout. The write
// cards sit beside the list, so leaving a stale supply next to a successful
// mint read as if the mint had done nothing.
func (a *App) refreshToken() {
	a.loadToken()
	if a.token.info == nil {
		return
	}
	if info, err := a.services.TokenInfo(strings.TrimSpace(a.token.tokenID)); err == nil {
		a.token.info = &info
	}
}

func (a *App) loadAllowance() {
	amt, err := a.services.TokenAllowance(
		strings.TrimSpace(a.token.tokenID), strings.TrimSpace(a.token.allowanceOwner),
		strings.TrimSpace(a.token.allowanceTo))
	if err != nil {
		a.token.status = errStatus(err)
		return
	}
	a.token.allowance = amt
	a.token.status = statusLine{}
}

// renderBurnDialog shows the confirmation while a burn is pending. MyGo draws
// it over the window; the burn runs only when the operator picks "Burn", the
// same gate the CLI's --confirm flag is.
func (a *App) renderBurnDialog(c *ui.Context) {
	if !a.token.askBurn {
		return
	}
	// Name the amount and the account in the question: a bare "Burn tokens?"
	// asked the operator to trust whichever fields the form held, and a wrong
	// amount is irreversible.
	detail := "Burning destroys the amount permanently. This cannot be undone."
	amount := strings.TrimSpace(a.token.burnAmount)
	from := strings.TrimSpace(a.token.burnFrom)
	if amount != "" || from != "" {
		detail = fmt.Sprintf(
			"Burn %s from %s? The tokens are destroyed permanently and cannot be recovered.",
			valueOr(amount), valueOr(from))
	}
	switch ui.AlertDialog(c, &a.token.askBurn, "Burn tokens?", detail, "Cancel", "Burn") {
	case 1:
		err := a.services.BurnToken(
			strings.TrimSpace(a.token.tokenID), from, amount, strings.TrimSpace(a.token.burnPriv))
		a.token.status = wrapErr(err)
		if err == nil {
			a.refreshToken()
		}
	}
}

func (a *App) tokenHistory(c *ui.Context) {
	formCard(c, "History", "Transfers this account sent or received.", func() {
		a.tokenHistoryBody(c)
	})
}

func (a *App) tokenHistoryBody(c *ui.Context) {
	fieldHint(c, "Owner (base64 public key)", "Leave empty to page the token's whole transfer stream.", &a.token.account)
	field(c, "Limit", &a.token.historyLimit)
	ui.Row(c).Children(func() {
		if secondaryButton(c, "Load History").Clicked() {
			limit := 50
			if s := strings.TrimSpace(a.token.historyLimit); s != "" {
				if n, err := strconv.Atoi(s); err == nil && n > 0 {
					limit = n
				}
			}
			rows, err := a.services.TokenHistory(strings.TrimSpace(a.token.tokenID), strings.TrimSpace(a.token.account), limit, 0)
			if err != nil {
				a.token.status = errStatus(err)
				a.token.history = nil
			} else {
				a.token.history = rows
				a.token.status = statusLine{}
			}
		}
	})
	if len(a.token.history) == 0 {
		emptyNote(c, "No transfers loaded yet.")
		return
	}
	hist := a.token.history
	ui.Table(c, &a.token.historyTable, []ui.TableColumn{
		{Title: "From", Width: 160},
		{Title: "To", Width: 160},
		{Title: "Amount", Width: 100, Align: ui.End},
		{Title: "Nonce", Width: 80, Align: ui.End},
	}, len(hist), func(row, col int) {
		h := hist[row]
		switch col {
		case 0:
			ui.Text(c, h.From).SingleLine()
		case 1:
			ui.Text(c, h.To).SingleLine()
		case 2:
			ui.Text(c, h.Amount)
		case 3:
			ui.Text(c, strconv.FormatUint(h.Nonce, 10))
		}
	})
}

// wrapErr turns a service call's error into the page's status line, and a nil
// error into a success line.
func wrapErr(err error) statusLine {
	if err != nil {
		return errStatus(err)
	}
	return okStatus("Done.")
}

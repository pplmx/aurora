package nativeui

import (
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
	mintTo       string
	mintAmount   string
	mintPriv     string
	xFrom        string
	xTo          string
	xAmount      string
	xPriv        string
	approveTo    string
	approveAmt   string
	approvePriv  string
	fromOwner    string
	fromSpender  string
	fromAmount   string
	fromPriv     string
	burnFrom     string
	burnAmount   string
	burnPriv     string
	allowanceTo  string
	historyLimit string

	allowance string
	history   []TransferRow

	loadErr error

	// askBurn drives the irreversible-burn confirmation. The burn only runs
	// when the operator picks the destructive button, mirroring the CLI's
	// --confirm gate.
	askBurn bool

	status statusLine

	historyTable ui.ListState
}

func (a *App) loadToken() {
	// The token page is form-driven: nothing loads until the operator asks for
	// a specific token, so a page switch only clears a stale error.
	a.token.loadErr = nil
}

func (a *App) tokenView(c *ui.Context) {
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().Gap(16).Children(func() {
			a.tokenCreate(c)
			a.tokenRead(c)
			a.tokenWrites(c)
			a.tokenHistory(c)
			a.renderBurnDialog(c)
		})
	})
}

func (a *App) tokenCreate(c *ui.Context) {
	sectionTitle(c, "Create Token", "Define supply and attach the owner key that will authorize future writes.")
	field(c, "Name", &a.token.name)
	field(c, "Symbol", &a.token.symbol)
	field(c, "Total Supply", &a.token.supply)
	fieldHint(c, "Decimals", "0 for a whole-unit token; 8 matches most asset conventions.", &a.token.decimals)
	fieldHint(c, "Owner (base64 public key)", "Required: the owner signs every later mint, transfer and burn.", &a.token.owner)
	if ui.PrimaryButton(c, "Create Token").Clicked() {
		a.createToken()
	}
	renderStatus(c, a.token.status)
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
	a.token.name, a.token.symbol, a.token.supply = "", "", ""
}

func (a *App) tokenRead(c *ui.Context) {
	ui.Divider(c)
	sectionTitle(c, "Inspect", "Read a token's metadata and an account's balance.")
	field(c, "Token ID", &a.token.tokenID)
	if ui.Button(c, "Get Token Info").Clicked() {
		t, err := a.services.TokenInfo(strings.TrimSpace(a.token.tokenID))
		if err != nil {
			a.token.status = errStatus(err)
			a.token.info = nil
		} else {
			a.token.info = &t
			a.token.status = statusLine{}
		}
	}
	if info := a.token.info; info != nil {
		ui.Text(c, info.Name+" ("+info.Symbol+")").FontSize(14).Bold()
		ui.Text(c, "Supply "+info.TotalSupply+" · decimals "+strconv.Itoa(info.Decimals)).FontSize(12)
		ui.Text(c, "Owner "+info.Owner).FontSize(12).TextColor(ui.Hex("#6b7c7f"))
	}

	field(c, "Account (base64 public key)", &a.token.account)
	if ui.Button(c, "Get Balance").Clicked() {
		bal, err := a.services.TokenBalance(strings.TrimSpace(a.token.tokenID), strings.TrimSpace(a.token.account))
		if err != nil {
			a.token.status = errStatus(err)
			a.token.balance = ""
		} else {
			a.token.balance = bal
			a.token.status = statusLine{}
		}
	}
	if a.token.balance != "" {
		ui.Text(c, "Balance: "+a.token.balance).FontSize(13).Bold()
	}
}

func (a *App) tokenWrites(c *ui.Context) {
	ui.Divider(c)
	sectionTitle(c, "Mint", "Only the token owner can mint, and only for a mintable token.")
	field(c, "To (base64 public key)", &a.token.mintTo)
	field(c, "Amount", &a.token.mintAmount)
	field(c, "Owner Private Key", &a.token.mintPriv)
	if ui.Button(c, "Mint").Clicked() {
		a.token.status = wrapErr(a.services.MintToken(
			strings.TrimSpace(a.token.tokenID), strings.TrimSpace(a.token.mintTo),
			strings.TrimSpace(a.token.mintAmount), strings.TrimSpace(a.token.mintPriv)))
	}

	ui.Divider(c)
	sectionTitle(c, "Transfer", "Move tokens from one account to another.")
	field(c, "From (base64 public key)", &a.token.xFrom)
	field(c, "To (base64 public key)", &a.token.xTo)
	field(c, "Amount", &a.token.xAmount)
	field(c, "From Private Key", &a.token.xPriv)
	if ui.Button(c, "Transfer").Clicked() {
		a.token.status = wrapErr(a.services.TransferToken(
			strings.TrimSpace(a.token.tokenID), strings.TrimSpace(a.token.xFrom),
			strings.TrimSpace(a.token.xTo), strings.TrimSpace(a.token.xAmount),
			strings.TrimSpace(a.token.xPriv)))
	}

	ui.Divider(c)
	sectionTitle(c, "Approve", "Let a spender move up to an amount on the owner's behalf.")
	field(c, "Spender (base64 public key)", &a.token.approveTo)
	field(c, "Amount", &a.token.approveAmt)
	field(c, "Owner Private Key", &a.token.approvePriv)
	if ui.Button(c, "Approve").Clicked() {
		a.token.status = wrapErr(a.services.ApproveToken(
			strings.TrimSpace(a.token.tokenID), strings.TrimSpace(a.token.xFrom),
			strings.TrimSpace(a.token.approveTo), strings.TrimSpace(a.token.approveAmt),
			strings.TrimSpace(a.token.approvePriv)))
	}

	ui.Divider(c)
	sectionTitle(c, "Transfer From", "Spend an allowance the owner granted.")
	field(c, "Owner (base64 public key)", &a.token.fromOwner)
	field(c, "To (base64 public key)", &a.token.xTo)
	field(c, "Amount", &a.token.fromAmount)
	field(c, "Spender Private Key", &a.token.fromPriv)
	if ui.Button(c, "Transfer From").Clicked() {
		a.token.status = wrapErr(a.services.TransferFromToken(
			strings.TrimSpace(a.token.tokenID), strings.TrimSpace(a.token.fromOwner),
			strings.TrimSpace(a.token.xTo), strings.TrimSpace(a.token.fromAmount),
			strings.TrimSpace(a.token.fromPriv)))
	}

	ui.Divider(c)
	sectionTitle(c, "Burn", "Destroy tokens from an account. This cannot be undone.")
	field(c, "From (base64 public key)", &a.token.burnFrom)
	field(c, "Amount", &a.token.burnAmount)
	field(c, "Owner Private Key", &a.token.burnPriv)
	if ui.Button(c, "Burn").Clicked() {
		// Burning is irreversible: the CLI requires --confirm, so the window
		// asks through a dialog instead of acting on a bare click.
		a.token.askBurn = true
	}

	ui.Divider(c)
	sectionTitle(c, "Allowance", "Read how much a spender may still move.")
	field(c, "Spender (base64 public key)", &a.token.allowanceTo)
	if ui.Button(c, "Get Allowance").Clicked() {
		amt, err := a.services.TokenAllowance(
			strings.TrimSpace(a.token.tokenID), strings.TrimSpace(a.token.xFrom),
			strings.TrimSpace(a.token.allowanceTo))
		if err != nil {
			a.token.status = errStatus(err)
		} else {
			a.token.allowance = amt
			a.token.status = statusLine{}
		}
	}
	if a.token.allowance != "" {
		ui.Text(c, "Allowance: "+a.token.allowance).FontSize(13).Bold()
	}
}

// renderBurnDialog shows the confirmation while a burn is pending. MyGo draws
// it over the window; the burn runs only when the operator picks "Burn", the
// same gate the CLI's --confirm flag is.
func (a *App) renderBurnDialog(c *ui.Context) {
	if !a.token.askBurn {
		return
	}
	switch ui.AlertDialog(c, &a.token.askBurn,
		"Burn tokens?",
		"Burning destroys the amount permanently. This cannot be undone.",
		"Cancel", "Burn") {
	case 1:
		a.token.status = wrapErr(a.services.BurnToken(
			strings.TrimSpace(a.token.tokenID), strings.TrimSpace(a.token.burnFrom),
			strings.TrimSpace(a.token.burnAmount), strings.TrimSpace(a.token.burnPriv)))
	}
}

func (a *App) tokenHistory(c *ui.Context) {
	ui.Divider(c)
	sectionTitle(c, "History", "Transfers this account sent or received.")
	fieldHint(c, "Owner (base64 public key)", "Leave empty to page the token's whole transfer stream.", &a.token.account)
	field(c, "Limit", &a.token.historyLimit)
	if ui.Button(c, "Load History").Clicked() {
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
	if len(a.token.history) == 0 {
		return
	}
	hist := a.token.history
	ui.Table(c, &a.token.historyTable, []ui.TableColumn{
		{Title: "From", Width: 160},
		{Title: "To", Width: 160},
		{Title: "Amount", Width: 100, Align: ui.End},
		{Title: "Nonce", Width: 70, Align: ui.End},
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

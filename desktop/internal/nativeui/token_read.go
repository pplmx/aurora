package nativeui

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *App) tokenRead(c *ui.Context) {
	formCard(c, "Inspect", "Read a token's metadata and an account's balance.", func() {
		a.tokenReadBody(c)
	})
}

func (a *App) tokenReadBody(c *ui.Context) {
	field(c, "Token ID", &a.token.tokenID)
	tokenID := strings.TrimSpace(a.token.tokenID)
	ui.Row(c).Children(func() {
		if secondaryButton(c, "Get Token Info").Disabled(!allSet(a.token.tokenID)).Clicked() {
			a.readTokenInfo()
		}
	})
	if a.token.infoErr != nil && a.token.infoLoadedID == tokenID {
		loadWarning(c, "Couldn't load token info", a.token.infoErr, a.token.info != nil)
	}
	if info := a.token.info; info != nil && a.token.infoLoadedID == tokenID {
		readout(c, []readoutRow{
			{"Name", info.Name + " (" + info.Symbol + ")"},
			{"Supply", info.TotalSupply},
			{"Decimals", strconv.Itoa(info.Decimals)},
			{"Owner", info.Owner},
		})
	}

	field(c, "Account (base64 public key)", &a.token.balanceAccount)
	account := strings.TrimSpace(a.token.balanceAccount)
	ui.Row(c).Children(func() {
		if secondaryButton(c, "Get Balance").Disabled(!allSet(a.token.tokenID, a.token.balanceAccount)).Clicked() {
			a.readTokenBalance()
		}
	})
	balanceMatches := a.token.balanceLoadedToken == tokenID && a.token.balanceLoadedOwner == account
	if a.token.balanceErr != nil && balanceMatches {
		loadWarning(c, "Couldn't load balance", a.token.balanceErr, a.token.balance != "")
	}
	if a.token.balance != "" && balanceMatches {
		readout(c, []readoutRow{{"Balance", a.token.balance}})
	}
}

// readTokenInfo runs the Inspect card's explicit read. The button is disabled
// until its input is present; this guard keeps a stale frame or a direct
// caller from sending an empty id to the service.
func (a *App) readTokenInfo() {
	id := strings.TrimSpace(a.token.tokenID)
	if rejectIncomplete(&a.token.status, "Token ID is required for this read.", id) {
		return
	}
	a.clearTokenInfoForAnotherSubject(id)
	if err := a.loadTokenInfo(id); err != nil {
		a.token.status = errStatus(err)
	} else {
		a.token.status = statusLine{}
	}
}

func (a *App) readTokenBalance() {
	id := strings.TrimSpace(a.token.tokenID)
	owner := strings.TrimSpace(a.token.balanceAccount)
	if rejectIncomplete(&a.token.status, "Token ID and account are required for this read.", id, owner) {
		return
	}
	a.clearTokenBalanceForAnotherRead(id, owner)
	if err := a.loadTokenBalance(id, owner); err != nil {
		a.token.status = errStatus(err)
	} else {
		a.token.status = statusLine{}
	}
}

// loadTokenInfo reads a token's metadata into the Inspect card. A failure
// keeps the rows only when they were loaded for the same token; the caller
// invalidates them when the subject changes.
func (a *App) loadTokenInfo(id string) error {
	subjectChanged := a.token.resolvedOwnerID != id
	info, err := a.services.TokenInfo(id)
	if err != nil {
		a.token.infoErr = err
		a.token.infoLoadedID = id
		return err
	}
	a.token.info = &info
	a.token.infoLoadedID = id
	a.token.infoErr = nil
	a.rememberTokenSubjectOwner(id, info.Owner)
	if subjectChanged {
		// Inspect is another entry point for choosing the token every write
		// acts on. Resolving an id outside the visible page must fill the same
		// owner fields as choosing its row, while a refresh of the same subject
		// must not erase values already entered in the forms.
		a.applyTokenSubject(info.Owner)
	}
	return nil
}

func (a *App) clearTokenInfoForAnotherSubject(id string) {
	if a.token.infoLoadedID == id {
		return
	}
	a.token.info = nil
	a.token.infoErr = nil
	a.token.infoLoadedID = ""
}

// loadTokenBalance reads one account's balance for one token. Both are part of
// the key: changing either makes the shown number describe something else.
func (a *App) loadTokenBalance(tokenID, owner string) error {
	balance, err := a.services.TokenBalance(tokenID, owner)
	if err != nil {
		a.token.balanceErr = err
		a.token.balanceLoadedToken, a.token.balanceLoadedOwner = tokenID, owner
		return err
	}
	a.token.balance = balance
	a.token.balanceLoadedToken, a.token.balanceLoadedOwner = tokenID, owner
	a.token.balanceErr = nil
	return nil
}

func (a *App) clearTokenBalanceForAnotherRead(tokenID, owner string) {
	if a.token.balanceLoadedToken == tokenID && a.token.balanceLoadedOwner == owner {
		return
	}
	a.token.balance = ""
	a.token.balanceErr = nil
	a.token.balanceLoadedToken, a.token.balanceLoadedOwner = "", ""
}

// loadTokenHistory reads one page of a token's transfers. The token, owner,
// limit and offset are the key; a failure keeps the rows only when they were
// produced by that same query.
func (a *App) loadTokenHistory(tokenID, owner string, limit, offset int) error {
	rows, err := a.services.TokenHistory(tokenID, owner, limit, offset)
	if err != nil {
		a.token.historyErr = err
		a.token.historyLoadedToken, a.token.historyLoadedOwner = tokenID, owner
		a.token.historyLoadedLimit, a.token.historyLoadedOffset = limit, offset
		return err
	}
	a.token.history = rows
	a.token.historyLoadedToken, a.token.historyLoadedOwner = tokenID, owner
	a.token.historyLoadedLimit, a.token.historyLoadedOffset = limit, offset
	a.token.historyErr = nil
	return nil
}

func (a *App) clearTokenHistoryForAnotherRead(tokenID, owner string, limit, offset int) {
	if a.token.historyLoadedToken == tokenID && a.token.historyLoadedOwner == owner &&
		a.token.historyLoadedLimit == limit && a.token.historyLoadedOffset == offset {
		return
	}
	a.token.history = nil
	a.token.historyErr = nil
	a.token.historyLoadedToken, a.token.historyLoadedOwner = "", ""
}

// selectedTokenOwner is the owner of the token the page is acting on, or "".
func (a *App) selectedTokenOwner() string {
	id := strings.TrimSpace(a.token.tokenID)
	if id != "" && a.token.resolvedOwnerID == id {
		return a.token.resolvedOwner
	}
	if id != "" && a.token.info != nil && a.token.infoLoadedID == id {
		return a.token.info.Owner
	}
	for _, token := range a.token.tokens {
		if token.ID == id {
			return token.Owner
		}
	}
	return ""
}

// tokenAllowance is deliberately outside the write tabs: it asks a read-only
// question, and keeping it on the left leaves the five mutating operations
// readable without overflowing their column.
func (a *App) tokenAllowance(c *ui.Context) {
	formCard(c, "Allowance", "Read how much a spender may still move.", func() {
		field(c, "Owner (base64 public key)", &a.token.allowanceOwner)
		field(c, "Spender (base64 public key)", &a.token.allowanceTo)
		ui.Row(c).Children(func() {
			if secondaryButton(c, "Get Allowance").Disabled(!allSet(a.token.tokenID, a.token.allowanceOwner, a.token.allowanceTo)).Clicked() {
				a.loadAllowance()
			}
		})
		matches := a.token.allowanceMatches(
			strings.TrimSpace(a.token.tokenID),
			strings.TrimSpace(a.token.allowanceOwner),
			strings.TrimSpace(a.token.allowanceTo),
		)
		if a.token.allowanceErr != nil && matches &&
			!loadWarning(c, "Couldn't load allowance", a.token.allowanceErr, a.token.allowance != "") {
			return
		}
		if a.token.allowance != "" && matches {
			readout(c, []readoutRow{{"Allowance", a.token.allowance}})
		}
	})
}

// tokenHistoryWindow is the page size and offset the History card sends: the
// defaults the form shows, clamped to the REST surface's ceiling.
func (a *App) tokenHistoryWindow() (limit, offset int) {
	limit = int(a.token.historyLimit)
	if limit < 1 {
		limit = 50
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	offset = int(a.token.historyOffset)
	if offset < 0 {
		offset = 0
	}
	if offset > maxListLimit {
		offset = maxListLimit
	}
	return limit, offset
}

func (a *App) loadAllowance() {
	tokenID := strings.TrimSpace(a.token.tokenID)
	owner := strings.TrimSpace(a.token.allowanceOwner)
	spender := strings.TrimSpace(a.token.allowanceTo)
	if rejectIncomplete(&a.token.status, "Token ID, owner, and spender are required for this read.", tokenID, owner, spender) {
		return
	}
	a.clearAllowanceForAnotherRead(tokenID, owner, spender)
	if err := a.loadTokenAllowance(tokenID, owner, spender); err != nil {
		a.token.status = errStatus(err)
		return
	}
	a.token.status = statusLine{}
}

func (a *App) loadTokenAllowance(tokenID, owner, spender string) error {
	amount, err := a.services.TokenAllowance(tokenID, owner, spender)
	a.token.allowanceLoadedToken = tokenID
	a.token.allowanceLoadedOwner = owner
	a.token.allowanceLoadedTo = spender
	if err != nil {
		a.token.allowanceErr = err
		return err
	}
	a.token.allowance = amount
	a.token.allowanceErr = nil
	return nil
}

func (a *App) clearAllowanceForAnotherRead(tokenID, owner, spender string) {
	if a.token.allowanceMatches(tokenID, owner, spender) {
		return
	}
	a.token.allowance = ""
	a.token.allowanceErr = nil
	a.token.allowanceLoadedToken = ""
	a.token.allowanceLoadedOwner = ""
	a.token.allowanceLoadedTo = ""
}

func (s tokenState) allowanceMatches(tokenID, owner, spender string) bool {
	return s.allowanceLoadedToken == tokenID &&
		s.allowanceLoadedOwner == owner &&
		s.allowanceLoadedTo == spender
}

func (a *App) tokenHistory(c *ui.Context) {
	formCard(c, "History", "Transfers for this token; enter an owner to narrow the stream to one account.", func() {
		a.tokenHistoryBody(c)
	})
}

func (a *App) tokenHistoryBody(c *ui.Context) {
	fieldHint(c, "Owner (base64 public key)", "Leave empty to page the token's whole transfer stream.", &a.token.historyOwner)
	ui.Collapsible(c, tr("Paging"), &a.token.historyAdvanced, func() {
		ui.Row(c).Gap(12).Children(func() {
			ui.Field(c, tr("Limit"), func() {
				ui.NumberInput(c, &a.token.historyLimit, 1, maxListLimit, 1)
			})
			ui.Field(c, tr("Offset"), func() {
				ui.NumberInput(c, &a.token.historyOffset, 0, maxListLimit, 1)
			})
		})
	})
	ui.Row(c).Children(func() {
		if secondaryButton(c, "Load History").Disabled(!allSet(a.token.tokenID)).Clicked() {
			a.readTokenHistory()
		}
	})
	id := strings.TrimSpace(a.token.tokenID)
	owner := strings.TrimSpace(a.token.historyOwner)
	limit, offset := a.tokenHistoryWindow()
	matches := id != "" && a.token.historyLoadedToken == id && a.token.historyLoadedOwner == owner &&
		a.token.historyLoadedLimit == limit && a.token.historyLoadedOffset == offset
	if a.token.historyErr != nil && matches &&
		!loadWarning(c, "Couldn't load history", a.token.historyErr, len(a.token.history) > 0) {
		return
	}
	if !matches {
		emptyNote(c, "No transfers loaded yet.")
		return
	}
	if len(a.token.history) == 0 {
		emptyNote(c, "No transfers found for this query.")
		return
	}
	collectionList(c, "Token transfer history",
		collectionRecords(a.token.history, transferCollectionRecord), -1, nil)
}

// transferCollectionRecord projects one recorded transfer onto a record. The
// amount is the title because it is what a reader scans the stream for; both
// accounts are fields so neither is cut to a prefix, and the nonce stays
// visible because it is what makes the transfer unique.
func transferCollectionRecord(transfer TransferRow) collectionRecord {
	return collectionRecord{
		title: valueOr(transfer.Amount),
		facts: []collectionFact{
			{label: "Amount", value: valueOr(transfer.Amount)},
			{label: "Nonce", value: strconv.FormatUint(transfer.Nonce, 10)},
		},
		fields: []collectionField{
			{label: "From", value: valueOr(transfer.From)},
			{label: "To", value: valueOr(transfer.To)},
		},
	}
}

func (a *App) readTokenHistory() {
	id := strings.TrimSpace(a.token.tokenID)
	if rejectIncomplete(&a.token.status, "Token ID is required for this read.", id) {
		return
	}
	owner := strings.TrimSpace(a.token.historyOwner)
	limit, offset := a.tokenHistoryWindow()
	a.clearTokenHistoryForAnotherRead(id, owner, limit, offset)
	if err := a.loadTokenHistory(id, owner, limit, offset); err != nil {
		a.token.status = errStatus(err)
	} else {
		a.token.status = statusLine{}
	}
}

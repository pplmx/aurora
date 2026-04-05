package lottery

import (
	"fmt"
	"strings"

	"github.com/pplmx/aurora/internal/blockchain"
	"github.com/rivo/tview"
)

type LotteryApp struct {
	app   *tview.Application
	chain *blockchain.BlockChain
	pages *tview.Pages
}

func NewLotteryApp() *LotteryApp {
	chain := blockchain.InitBlockChain()

	app := tview.NewApplication()
	pages := tview.NewPages()

	return &LotteryApp{
		app:   app,
		chain: chain,
		pages: pages,
	}
}

func (a *LotteryApp) Run() error {
	a.setupPages()
	a.app.SetRoot(a.pages, true)
	return a.app.Run()
}

func (a *LotteryApp) setupPages() {
	menu := tview.NewModal()
	menu.SetText("VRF Lottery System\n\nSelect operation:")
	menu.AddButtons([]string{"Create Lottery", "View History", "Exit"})
	menu.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		switch buttonLabel {
		case "Create Lottery":
			a.app.SetRoot(a.createLotteryPage(), true)
		case "View History":
			a.app.SetRoot(a.historyPage(), true)
		case "Exit":
			a.app.Stop()
		}
	})

	a.pages.AddPage("menu", menu, true, true)
	a.app.SetRoot(menu, true)
}

func (a *LotteryApp) createLotteryPage() tview.Primitive {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBorder(true).SetTitle("Create New Lottery")

	helpText := tview.NewTextView()
	helpText.SetText("Enter participant names (one per line) and random seed")
	helpText.SetDynamicColors(true)
	flex.AddItem(helpText, 2, 0, false)

	participantsLabel := tview.NewTextView()
	participantsLabel.SetText("Participants List:")
	flex.AddItem(participantsLabel, 1, 0, false)

	participantsInput := tview.NewTextArea()
	participantsInput.SetPlaceholder("Alice\nBob\nCharlie\nDavid")
	flex.AddItem(participantsInput, 8, 0, false)

	seedLabel := tview.NewTextView()
	seedLabel.SetText("Random Seed:")
	flex.AddItem(seedLabel, 1, 0, false)

	seedInput := tview.NewInputField()
	seedInput.SetPlaceholder("Enter random seed...")
	flex.AddItem(seedInput, 2, 0, false)

	countLabel := tview.NewTextView()
	countLabel.SetText("Number of Winners:")
	flex.AddItem(countLabel, 1, 0, false)

	countInput := tview.NewInputField()
	countInput.SetPlaceholder("3")
	countInput.SetText("3")
	flex.AddItem(countInput, 2, 0, false)

	buttonFlex := tview.NewFlex()

	createBtn := tview.NewButton("[Create Lottery]")
	createBtn.SetSelectedFunc(func() {
		participants := parseTextArea(participantsInput.GetText())
		seed := seedInput.GetText()
		count := 3
		fmt.Sscanf(countInput.GetText(), "%d", &count)

		if len(participants) < count || seed == "" {
			a.showError("Participants and seed cannot be empty, and participants must be more than winners")
			return
		}

		result := a.runLottery(participants, seed, count)
		a.app.SetRoot(a.resultPage(result), true)
	})
	buttonFlex.AddItem(createBtn, 0, 1, false)

	backBtn := tview.NewButton("[Back]")
	backBtn.SetSelectedFunc(func() {
		a.app.SetRoot(a.pages, true)
	})
	buttonFlex.AddItem(backBtn, 0, 1, false)

	flex.AddItem(buttonFlex, 3, 0, false)

	return flex
}

func (a *LotteryApp) historyPage() tview.Primitive {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBorder(true).SetTitle("Lottery History")

	textView := tview.NewTextView()
	textView.SetDynamicColors(true)
	textView.SetWrap(true)

	records := a.chain.GetLotteryRecords()
	if len(records) == 0 {
		fmt.Fprintln(textView, "No lottery records found")
	} else {
		for i, data := range records {
			fmt.Fprintf(textView, "--- Lottery #%d ---\n%s\n\n", i+1, data)
		}
	}

	flex.AddItem(textView, 0, 1, false)

	backBtn := tview.NewButton("[Back]")
	backBtn.SetSelectedFunc(func() {
		a.app.SetRoot(a.pages, true)
	})
	flex.AddItem(backBtn, 3, 0, false)

	return flex
}

func (a *LotteryApp) resultPage(record *LotteryRecord) tview.Primitive {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBorder(true).SetTitle("Lottery Result")

	textView := tview.NewTextView()
	textView.SetDynamicColors(true)

	fmt.Fprintln(textView, "Lottery completed!")
	fmt.Fprintln(textView, "")
	fmt.Fprintf(textView, "Lottery ID: %s\n", record.ID)
	fmt.Fprintf(textView, "Block Height: #%d\n\n", record.BlockHeight)

	fmt.Fprintln(textView, "Winners:")
	for i, w := range record.Winners {
		fmt.Fprintf(textView, "   %d. %s (%s)\n", i+1, w, record.WinnerAddrs[i])
	}

	fmt.Fprintln(textView, "")
	vrfOut := record.VRFOutput
	vrfProof := record.VRFProof
	if len(vrfOut) > 32 {
		vrfOut = vrfOut[:32]
	}
	if len(vrfProof) > 32 {
		vrfProof = vrfProof[:32]
	}
	fmt.Fprintf(textView, "VRF Output: %s...\n", vrfOut)
	fmt.Fprintf(textView, "VRF Proof: %s...\n", vrfProof)

	flex.AddItem(textView, 0, 1, false)

	backBtn := tview.NewButton("[Back to Main Menu]")
	backBtn.SetSelectedFunc(func() {
		a.app.SetRoot(a.pages, true)
	})
	flex.AddItem(backBtn, 3, 0, false)

	return flex
}

func (a *LotteryApp) showError(msg string) {
	modal := tview.NewModal()
	modal.SetText(msg)
	modal.AddButtons([]string{"OK"})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		a.app.SetRoot(a.createLotteryPage(), true)
	})
	a.app.SetRoot(modal, true)
}

func (a *LotteryApp) runLottery(participants []string, seed string, count int) *LotteryRecord {
	pk, sk, _ := GenerateKeyPair()
	output, proof, _ := VRFProve(sk, []byte(seed))

	winners := SelectWinners(output, participants, count)
	winnerAddrs := make([]string, len(winners))
	for i, w := range winners {
		winnerAddrs[i] = NameToAddress(w)
	}

	record := CreateLotteryRecord(seed, participants, winners, winnerAddrs, output, proof, 0)

	jsonData, _ := record.ToJSON()
	height, _ := a.chain.AddLotteryRecord(jsonData)
	record.BlockHeight = height

	_ = pk
	return record
}

func parseTextArea(text string) []string {
	var result []string
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

package nativeui

// page identifies the module the window is showing.
type page int

const (
	pageDashboard page = iota
	pageLottery
	pageVoting
	pageToken
	pageOracle
	pageNFT
	pageBlockchain
	pageSettings
)

// pageIDs are the stable identifiers the sidebar chooses by. They are
// separate from the titles because a title is shown text: translating it
// later must not change what a choice is remembered as.
var pageIDs = map[page]string{
	pageDashboard:  "dashboard",
	pageLottery:    "lottery",
	pageVoting:     "voting",
	pageToken:      "token",
	pageOracle:     "oracle",
	pageNFT:        "nft",
	pageBlockchain: "blockchain",
	pageSettings:   "settings",
}

// pageByID is the reverse lookup the sidebar needs when a click arrives.
func pageByID(id string) (page, bool) {
	for p, want := range pageIDs {
		if want == id {
			return p, true
		}
	}
	return 0, false
}

var pageTitles = map[page]string{
	pageDashboard:  "Dashboard",
	pageLottery:    "Lottery",
	pageVoting:     "Voting",
	pageToken:      "Token",
	pageOracle:     "Oracle",
	pageNFT:        "NFT",
	pageBlockchain: "Blockchain",
	pageSettings:   "Settings",
}

// moduleOrder is the module section of the sidebar. pageOrder is the full
// keyboard order: the seven modules, then Settings in the Application section.
var moduleOrder = []page{
	pageDashboard, pageLottery, pageVoting, pageToken, pageOracle, pageNFT, pageBlockchain,
}

var pageOrder = []page{
	pageDashboard, pageLottery, pageVoting, pageToken, pageOracle, pageNFT, pageBlockchain, pageSettings,
}

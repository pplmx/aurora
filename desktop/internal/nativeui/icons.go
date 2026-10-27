package nativeui

import "github.com/egoist/mygo/ui"

// The sidebar's icons, one per page.
//
// They are Lucide geometry (ISC), drawn as strokes in currentColor so an icon
// takes the colour of the row it sits in: the accent while its page is
// chosen, muted otherwise. The reference MyGo apps (GoRex, the NetEase client)
// give every module a glyph for the same reason: a column of labels alone is
// slower to scan than a column of labelled glyphs, and the sidebar is the one
// control that is always on screen.
//
// The SVG data lives here rather than in files so the desktop module needs no
// embed directive and the icons travel with the package's tests.
const svgOpen = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">`

var (
	dashboardIcon = ui.MustParseSVG([]byte(svgOpen +
		`<rect x="3" y="3" width="7" height="9" rx="1"/>` +
		`<rect x="14" y="3" width="7" height="5" rx="1"/>` +
		`<rect x="14" y="12" width="7" height="9" rx="1"/>` +
		`<rect x="3" y="16" width="7" height="5" rx="1"/>` +
		`</svg>`))
	lotteryIcon = ui.MustParseSVG([]byte(svgOpen +
		`<path d="M2 9a3 3 0 0 1 0 6v2a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-2a3 3 0 0 1 0-6V7a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2Z"/>` +
		`<path d="M13 5v2"/><path d="M13 11v2"/><path d="M13 17v2"/>` +
		`</svg>`))
	votingIcon = ui.MustParseSVG([]byte(svgOpen +
		`<path d="M5 3h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Z"/>` +
		`<path d="m9 12 2 2 4-4"/>` +
		`</svg>`))
	tokenIcon = ui.MustParseSVG([]byte(svgOpen +
		`<circle cx="8" cy="8" r="6"/>` +
		`<path d="M18.09 10.37A6 6 0 1 1 10.34 18"/>` +
		`<path d="M7 6h1v4"/>` +
		`<path d="m16.71 13.88.7.71-2.82 2.82"/>` +
		`</svg>`))
	oracleIcon = ui.MustParseSVG([]byte(svgOpen +
		`<path d="M4.9 19.1C1 15.2 1 8.8 4.9 4.9"/>` +
		`<path d="M7.8 16.2c-2.3-2.3-2.3-6.1 0-8.5"/>` +
		`<circle cx="12" cy="12" r="2"/>` +
		`<path d="M16.2 7.8c2.3 2.3 2.3 6.1 0 8.5"/>` +
		`<path d="M19.1 4.9C23 8.8 23 15.2 19.1 19.1"/>` +
		`</svg>`))
	nftIcon = ui.MustParseSVG([]byte(svgOpen +
		`<rect x="3" y="3" width="18" height="18" rx="2"/>` +
		`<circle cx="9" cy="9" r="2"/>` +
		`<path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/>` +
		`</svg>`))
	blockchainIcon = ui.MustParseSVG([]byte(svgOpen +
		`<rect x="3" y="3" width="7" height="7" rx="1"/>` +
		`<rect x="14" y="3" width="7" height="7" rx="1"/>` +
		`<rect x="8" y="14" width="7" height="7" rx="1"/>` +
		`<path d="M6.5 10v2.5a1 1 0 0 0 1 1H8"/>` +
		`<path d="M17.5 10v2.5a1 1 0 0 1-1 1H16"/>` +
		`</svg>`))
	settingsIcon = ui.MustParseSVG([]byte(svgOpen +
		`<path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.09a2 2 0 0 1 1 1.74v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.38a2 2 0 0 0-.73-2.73l-.15-.09a2 2 0 0 1-1-1.74v-.51a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2Z"/>` +
		`<circle cx="12" cy="12" r="3"/>` +
		`</svg>`))
)

// pageIcons maps every page to its glyph. The map is checked by a test against
// pageOrder so a page added without an icon cannot render a blank row.
var pageIcons = map[page]*ui.SVG{
	pageDashboard:  dashboardIcon,
	pageLottery:    lotteryIcon,
	pageVoting:     votingIcon,
	pageToken:      tokenIcon,
	pageOracle:     oracleIcon,
	pageNFT:        nftIcon,
	pageBlockchain: blockchainIcon,
	pageSettings:   settingsIcon,
}

// Action glyphs for the command palette. They use the same Lucide stroke
// language as the page icons.
var (
	refreshIcon = ui.MustParseSVG([]byte(svgOpen +
		`<path d="M21 12a9 9 0 0 0-15.5-6.36L3 8"/>` +
		`<path d="M3 3v5h5"/>` +
		`<path d="M3 12a9 9 0 0 0 15.5 6.36L21 16"/>` +
		`<path d="M16 16h5v5"/>` +
		`</svg>`))
	backupIcon = ui.MustParseSVG([]byte(svgOpen +
		`<ellipse cx="12" cy="5" rx="9" ry="3"/>` +
		`<path d="M3 5v14a9 3 0 0 0 18 0V5"/>` +
		`<path d="M3 12a9 3 0 0 0 18 0"/>` +
		`</svg>`))
	exportIcon = ui.MustParseSVG([]byte(svgOpen +
		`<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>` +
		`<path d="m17 8-5-5-5 5"/>` +
		`<path d="M12 3v12"/>` +
		`</svg>`))
	importIcon = ui.MustParseSVG([]byte(svgOpen +
		`<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>` +
		`<path d="m7 10 5 5 5-5"/>` +
		`<path d="M12 15V3"/>` +
		`</svg>`))
)

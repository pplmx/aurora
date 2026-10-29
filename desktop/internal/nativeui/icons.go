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
	searchIcon = ui.MustParseSVG([]byte(svgOpen +
		`<circle cx="11" cy="11" r="8"/>` +
		`<path d="m21 21-4.3-4.3"/>` +
		`</svg>`))
	usersIcon = ui.MustParseSVG([]byte(svgOpen +
		`<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/>` +
		`<circle cx="9" cy="7" r="4"/>` +
		`<path d="M22 21v-2a4 4 0 0 0-3-3.87"/>` +
		`<path d="M16 3.13a4 4 0 0 1 0 7.75"/>` +
		`</svg>`))
	userCheckIcon = ui.MustParseSVG([]byte(svgOpen +
		`<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/>` +
		`<circle cx="9" cy="7" r="4"/>` +
		`<path d="m16 11 2 2 4-4"/>` +
		`</svg>`))
	calendarIcon = ui.MustParseSVG([]byte(svgOpen +
		`<rect x="3" y="4" width="18" height="18" rx="2"/>` +
		`<path d="M16 2v4"/>` +
		`<path d="M8 2v4"/>` +
		`<path d="M3 10h18"/>` +
		`</svg>`))
	sparkIcon = ui.MustParseSVG([]byte(svgOpen +
		`<path d="M9.94 4.06a.5.5 0 0 1 .95 0l1.3 4.02a2 2 0 0 0 1.27 1.27l4.02 1.3a.5.5 0 0 1 0 .95l-4.02 1.3a2 2 0 0 0-1.27 1.27l-1.3 4.02a.5.5 0 0 1-.95 0l-1.3-4.02a2 2 0 0 0-1.27-1.27l-4.02-1.3a.5.5 0 0 1 0-.95l4.02-1.3a2 2 0 0 0 1.27-1.27z"/>` +
		`<path d="M19 3v4"/>` +
		`<path d="M21 5h-4"/>` +
		`</svg>`))
	shieldCheckIcon = ui.MustParseSVG([]byte(svgOpen +
		`<path d="M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z"/>` +
		`<path d="m9 12 2 2 4-4"/>` +
		`</svg>`))
	arrowRightIcon = ui.MustParseSVG([]byte(svgOpen +
		`<path d="M5 12h14"/>` +
		`<path d="m12 5 7 7-7 7"/>` +
		`</svg>`))
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
	// warningIcon marks a confirmation whose action cannot be undone, and a
	// load that kept stale rows. The triangle with an exclamation is the
	// shape readers already know from every desktop alert.
	warningIcon = ui.MustParseSVG([]byte(svgOpen +
		`<path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"/>` +
		`<path d="M12 9v4"/>` +
		`<path d="M12 17h.01"/>` +
		`</svg>`))
)

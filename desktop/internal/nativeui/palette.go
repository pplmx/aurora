package nativeui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
)

// paletteEntry is one row in the Ctrl/Cmd+K switcher: either a page or a
// command. The action is nil for pages; navigation and side effects share the
// same selection model so Enter never means two different things.
type paletteEntry struct {
	title  string
	detail string
	icon   *ui.SVG
	page   page
	// accent, when hasAccent is set, is the page whose hue the icon takes. A
	// record sets it so a list mixing two modules is scannable by colour the way
	// the ledger's rows are. The hue is resolved per frame rather than stored,
	// because which palette answers depends on the appearance being painted.
	// Pages and commands leave it unset and use the accent: pageDashboard is the
	// zero page, so there is no spare value to mean "none".
	accent    page
	hasAccent bool
	// matched marks an entry the storage layer already filtered. The palette's
	// own text filter runs over what it draws, and a record's headline is its
	// module and action — the query that found it may appear only in the payload
	// the search read, which is not drawn at all. Such an entry is accepted as it
	// came back instead of being filtered a second time by text it does not show.
	matched bool
	// action receives the frame's context so an entry can copy to the
	// clipboard; navigating entries leave it nil.
	action func(*ui.Context)
}

// commandPalette lets a keyboard-first operator jump to a page or run a file
// action without leaving the current form. Dialogs keep their own confirmation
// model; the palette only invokes the same callbacks as the File menu.
func (a *App) commandPalette(c *ui.Context) {
	width, height := c.Size()
	panelWidth := min(520*textScale(c), width-100)
	panel := ui.Column(c).Width(panelWidth).MaxHeight(height-40).Padding(space4+2).Gap(space2+2).Radius(radiusPanel).
		Background(c.Theme().Surface).BorderWidth(1).BorderColor(c.Theme().Border).
		Shadow(0, 18, 48, -8, ui.RGBA(0, 0, 0, 0.5))
	// A lit top edge ties the palette to the accent that opened it, and marks
	// it as the one floating surface above the page rather than another card.
	accentHairline(c, panel, c.Theme().Accent)
	panel.Children(func() {
		heading(c, tr("Jump to"), 17, 2)
		search := searchField(c, &a.paletteQuery, "Search pages and commands").AutoFocus().
			Description(tr("Paste a record id as kind:id to open it, for example token:…"))
		submitted := search.Submitted()
		queryChanged := a.paletteQuery != a.paletteApplied
		if queryChanged {
			a.paletteApplied = a.paletteQuery
			a.paletteChoice = 0
			a.paletteScroll.Y = 0
		}
		matches := a.paletteMatches()
		announceSearchResults(c, queryChanged, len(matches))
		if len(matches) == 0 {
			emptyNote(c, fmt.Sprintf(tr("No matches for “%s”."), strings.TrimSpace(a.paletteQuery)))
			return
		}
		if a.paletteChoice >= len(matches) {
			a.paletteChoice = len(matches) - 1
		}
		if a.paletteChoice < 0 {
			a.paletteChoice = 0
		}
		reveal := a.paletteReveal
		a.paletteReveal = false
		list := ui.Scroll(c).FillWidth().Gap(10).
			MaxHeight(paletteListHeight(c, len(matches))).
			TrackScroll(&a.paletteScroll).Label(tr("Pages and commands"))
		list.Children(func() {
			for i, entry := range matches {
				row, clicked := paletteRow(c, entry, i == a.paletteChoice)
				if i == a.paletteChoice {
					// Keep the input focused for typing while assistive
					// technology follows the row the arrows selected.
					search.ActiveDescendant(row)
					if reveal {
						row.ScrollIntoView()
					}
				}
				if clicked || (submitted && i == a.paletteChoice) {
					a.paletteOpen = false
					if entry.action != nil {
						entry.action(c)
					} else {
						a.setPage(entry.page)
					}
					return
				}
			}
		})
		// The palette is the one surface a reader opens without a pointer;
		// printing its three keys here means the arrow/Enter/Escape model is
		// learned where it is used instead of only in the Settings reference.
		// Each chord gets a key cap so the three read as keys rather than as a
		// sentence of arrows.
		ui.Row(c).FillWidth().Height(1).Background(c.Theme().Border)
		ui.Row(c).FillWidth().Wrap().AlignItems(ui.Center).Gap(space2).Children(func() {
			for i, hint := range []struct{ key, label string }{
				{"↑↓", "move"}, {"Enter", "open"}, {"Esc", "close"},
			} {
				if i > 0 {
					ui.Text(c, "·").FontSize(textSize(c, 10)).TextColor(c.Theme().TextMuted).Shrink(0)
				}
				shortcutCap(c, hint.key)
				ui.Text(c, tr(hint.label)).FontSize(textSize(c, 10)).
					TextColor(c.Theme().TextMuted).SingleLine().Shrink(0)
			}
		})
	})
	// Arrow keys move the highlight; Enter is handled by SearchField above.
	if panel.OverlayShortcut(0, ui.KeyDown) {
		a.paletteChoice++
		a.paletteReveal = true
		c.Invalidate()
	}
	if panel.OverlayShortcut(0, ui.KeyUp) {
		a.paletteChoice--
		a.paletteReveal = true
		c.Invalidate()
	}
}

func (a *App) paletteMatches() []paletteEntry {
	entries := make([]paletteEntry, 0, len(pageOrder)+4)
	for _, p := range pageOrder {
		entries = append(entries, paletteEntry{
			title: pageTitles[p], detail: pagePurpose[p], icon: pageIcons[p], page: p,
		})
	}
	// Window commands come before file work: a reader who types "zoom" or
	// "refresh" is steering the window, and the list reads in the order those
	// belong to. Navigation stays live while a background write owns the
	// database, but reload and file commands must not start a second repository
	// operation.
	//
	// Zoom is the one exception to that gate, and it stays above the busy check
	// to say so: it reads and writes no database, and a reader waiting on a long
	// import may well want the page larger. Its description carries the current
	// ratio, so opening the palette answers "how big is it now" without
	// Settings.
	// A page that could not be read belongs here, and near the top: the palette
	// is where a reader goes when they cannot find something, and "the page is
	// failing" is the answer to a surprising number of those searches. It names
	// the page so the entry reads as a destination rather than a warning.
	if failures := a.loadFailures(); len(failures) > 0 {
		target := pageTitles[failures[0].Page]
		detail := fmt.Sprintf(tr("Open %s, which could not be read"), tr(target))
		if len(failures) > 1 {
			detail = fmt.Sprintf(tr("Open %s; %d pages could not be read"), tr(target), len(failures))
		}
		entries = append(entries, paletteEntry{
			title: fmt.Sprintf(tr("Open %s"), tr(target)), detail: detail,
			icon: warningIcon, action: func(*ui.Context) { a.openFirstFailure() },
		})
	}

	zoom := zoomLabel(a.contentZoom)
	entries = append(entries,
		paletteEntry{
			title: "Zoom in", detail: fmt.Sprintf(tr("Enlarge the window content · now %s"), zoom),
			icon: zoomInIcon, action: func(*ui.Context) { a.setContentZoom(nextZoom(a.contentZoom, +1)) },
		},
		paletteEntry{
			title: "Zoom out", detail: fmt.Sprintf(tr("Shrink the window content · now %s"), zoom),
			icon: zoomOutIcon, action: func(*ui.Context) { a.setContentZoom(nextZoom(a.contentZoom, -1)) },
		},
		paletteEntry{
			title: "Actual size", detail: fmt.Sprintf(tr("Reset the content zoom to 100% · now %s"), zoom),
			icon: zoomResetIcon, action: func(*ui.Context) { a.setContentZoom(1) },
		},
	)
	if a.busy == "" {
		if a.page != pageSettings {
			entries = append(entries, paletteEntry{
				title: "Refresh current page", detail: "Refresh the data shown on this page",
				icon: refreshIcon, action: func(*ui.Context) { a.refresh() },
			})
		}
		if a.actions.BackUpDatabase != nil {
			entries = append(entries, paletteEntry{
				title: "Back up database…", detail: "Create a WAL-complete snapshot",
				icon: backupIcon, action: func(*ui.Context) { a.actions.BackUpDatabase() },
			})
		}
		if a.actions.ExportLotteries != nil {
			entries = append(entries, paletteEntry{
				title: "Export lotteries…", detail: "Write the portable lottery document",
				icon: exportIcon, action: func(*ui.Context) { a.actions.ExportLotteries() },
			})
		}
		if a.actions.ImportLotteries != nil {
			entries = append(entries, paletteEntry{
				title: "Import lotteries…", detail: "Load a portable lottery document",
				icon: importIcon, action: func(*ui.Context) { a.actions.ImportLotteries() },
			})
		}
		if a.actions.ShowDataFolder != nil {
			entries = append(entries, paletteEntry{
				title: "Show data folder", detail: "Open the folder that holds the database and backups",
				icon: folderIcon, action: func(*ui.Context) { a.actions.ShowDataFolder() },
			})
		}
	}
	// A pasted "kind:id" is a jump, not a search: the id an operator copies into
	// a ticket is exactly the thing they want to open, and searching for it as
	// text would only ever match the command that pasted it. The entry comes
	// first so Enter on a pasted id does what the reader meant, without knowing
	// which page owns it.
	var jump *paletteEntry
	if target, id, ok := parseRecordQuery(a.paletteQuery); ok {
		jump = &paletteEntry{
			title:  fmt.Sprintf(tr("Open %s"), tr(pageTitles[target])),
			detail: id,
			icon:   pageIcons[target],
			action: func(*ui.Context) { a.ShowRecord(pageIDs[target], id) },
		}
	}
	// Records that match the query come after the pages and commands: they are
	// what an operator investigating a problem is usually after, but they are
	// also the most numerous, and a page or command whose title starts with the
	// query should still rank first.
	entries = append(entries, a.paletteRecordEntries(a.paletteQuery)...)
	// Read-only and always available: a support reply should be copyable while
	// a background write owns the database.
	if a.actions.Version != "" {
		entries = append(entries, paletteEntry{
			title:  "Copy Diagnostics",
			detail: "Put the build, database and ledger facts on the clipboard",
			icon:   shieldCheckIcon, action: a.copyDiagnostics,
		})
	}
	matches := entries[:0]
	// A parsed jump is offered on its own: the literal text "token:<id>" matches
	// no page or command, so passing it through the filter below would drop the
	// one entry the reader asked for.
	if jump != nil {
		return []paletteEntry{*jump}
	}
	for _, entry := range entries {
		if entry.matched {
			matches = append(matches, entry)
			continue
		}
		// Search what the palette shows, not only the English source strings.
		// A Chinese window renders translated titles and purposes, so a query
		// copied from the screen must match the same text.
		if containsFold(a.paletteQuery,
			entry.title, tr(entry.title), entry.detail, tr(entry.detail)) {
			matches = append(matches, entry)
		}
	}
	// Order by where the query hits, not by sidebar order: typing "t" used to
	// list Lottery and Voting before Token because they come first on the
	// sidebar. A stable sort keeps the page/command order inside each group.
	query := strings.TrimSpace(a.paletteQuery)
	sort.SliceStable(matches, func(i, j int) bool {
		return paletteRank(matches[i], query) < paletteRank(matches[j], query)
	})
	return matches
}

// paletteRank scores where a query matches an entry, lowest first: a title
// that starts with it, then a title that contains it, then a description.
func paletteRank(entry paletteEntry, query string) int {
	query = strings.ToLower(query)
	if query == "" {
		return 0
	}
	title := strings.ToLower(entry.title)
	translated := strings.ToLower(tr(entry.title))
	switch {
	case strings.HasPrefix(title, query), strings.HasPrefix(translated, query):
		return 0
	case strings.Contains(title, query), strings.Contains(translated, query):
		return 1
	default:
		return 2
	}
}

func paletteRow(c *ui.Context, entry paletteEntry, selected bool) (ui.Element, bool) {
	title := tr(entry.title)
	b := ui.ButtonBase(c).FillWidth().Height(paletteRowHeight(c)).PaddingX(space2 + 2).Radius(radiusSmall).
		AlignItems(ui.Center).Label(title).Description(tr(entry.detail)).
		TextColor(c.Theme().Text).Selected(selected)
	fill := ui.Transparent
	if selected {
		fill = c.Theme().Selection
	}
	if b.Hovered() || b.Pressed() {
		fill = c.Theme().SurfaceHover
	}
	b.Background(fill)
	b.Children(func() {
		ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
			size := 18 * textScale(c)
			tone := c.Theme().Accent
			if entry.hasAccent {
				tone = pageAccent(c, entry.accent)
			}
			ui.Icon(c, entry.icon).Size(size, size).TextColor(tone)
			ui.Column(c).Gap(1).Children(func() {
				ui.Text(c, title).FontSize(textSize(c, 13)).Bold().TextColor(c.Theme().Text).SingleLine()
				ui.Text(c, tr(entry.detail)).FontSize(textSize(c, 11)).TextColor(c.Theme().TextMuted).SingleLine()
			})
		})
	})
	return b, b.Clicked()
}

func paletteRowHeight(c *ui.Context) float32 {
	return 48 * textScale(c)
}

func paletteListHeight(c *ui.Context, rows int) float32 {
	if rows <= 0 {
		return 0
	}
	_, height := c.Size()
	rowHeight := paletteRowHeight(c)
	contentHeight := float32(rows)*rowHeight + float32(rows-1)*10
	availableHeight := max(rowHeight, height-190)
	return min(contentHeight, availableHeight)
}

// openPalette resets the switcher and shows it. Both the keyboard chord and
// the sidebar button come through here so a remembered query never reopens a
// dialog the operator did not type.
func (a *App) openPalette() {
	a.paletteQuery = ""
	a.paletteApplied = ""
	a.paletteChoice = 0
	a.paletteScroll.Y = 0
	a.paletteReveal = false
	a.paletteOpen = true
}

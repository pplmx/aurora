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
	action func()
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
		search := searchField(c, &a.paletteQuery, "Search pages and commands").AutoFocus()
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
						entry.action()
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
	// Navigation stays live while a background write owns the database, but
	// reload and file commands must not start a second repository operation.
	if a.busy == "" {
		if a.page != pageSettings {
			entries = append(entries, paletteEntry{
				title: "Refresh current page", detail: "Reload the data shown on this page",
				icon: refreshIcon, action: a.refresh,
			})
		}
		if a.actions.BackUpDatabase != nil {
			entries = append(entries, paletteEntry{
				title: "Back up database…", detail: "Create a WAL-complete snapshot",
				icon: backupIcon, action: a.actions.BackUpDatabase,
			})
		}
		if a.actions.ExportLotteries != nil {
			entries = append(entries, paletteEntry{
				title: "Export lotteries…", detail: "Write the portable lottery document",
				icon: exportIcon, action: a.actions.ExportLotteries,
			})
		}
		if a.actions.ImportLotteries != nil {
			entries = append(entries, paletteEntry{
				title: "Import lotteries…", detail: "Load a portable lottery document",
				icon: importIcon, action: a.actions.ImportLotteries,
			})
		}
		if a.actions.ShowDataFolder != nil {
			entries = append(entries, paletteEntry{
				title: "Show data folder", detail: "Open the folder that holds the database and backups",
				icon: folderIcon, action: a.actions.ShowDataFolder,
			})
		}
	}
	matches := entries[:0]
	for _, entry := range entries {
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
			ui.Icon(c, entry.icon).Size(size, size).TextColor(c.Theme().Accent)
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

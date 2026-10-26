package deskapp

import (
	"strconv"
	"testing"

	"github.com/egoist/mygo"
	"github.com/pplmx/aurora/desktop/internal/nativeui"
	"github.com/stretchr/testify/require"
)

// TestGoMenuOffersEveryPage pins the menu bar's page list: it carries the same
// titles in the same order as the sidebar, one number accelerator each, and
// choosing one calls the window's ShowPage with that title. Without the menu
// the Ctrl/Cmd+1..7 shortcuts existed but nothing in the window told anyone.
func TestGoMenuOffersEveryPage(t *testing.T) {
	var shown []string
	menu := NewMenu(About{}, func(page string) { shown = append(shown, page) }, nil)

	var goItem *mygo.MenuItem
	for _, item := range menu.Items() {
		if item.Label == T("desktop.menu.go") {
			goItem = item
			break
		}
	}
	require.NotNil(t, goItem, "the menu bar must carry a Go menu")

	titles := nativeui.PageTitles()
	require.Len(t, goItem.Submenu, len(titles))
	for i, item := range goItem.Submenu {
		require.Equal(t, titles[i], item.Label, "page %d must use the sidebar title", i)
		require.Equal(t, "CmdOrCtrl+"+strconv.Itoa(i+1), item.Accelerator)
	}

	goItem.Submenu[1].Click(goItem.Submenu[1], nil)
	require.Equal(t, []string{titles[1]}, shown)
}

// TestGoMenuNeedsAPageSwitcher keeps NewMenu usable before the window state
// exists: without a callback the Go menu is simply absent, not a dead menu.
func TestGoMenuNeedsAPageSwitcher(t *testing.T) {
	menu := NewMenu(About{}, nil, nil)
	for _, item := range menu.Items() {
		require.NotEqual(t, T("desktop.menu.go"), item.Label,
			"a Go menu with no way to switch pages must not appear")
	}
}

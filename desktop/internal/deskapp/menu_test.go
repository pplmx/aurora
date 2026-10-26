package deskapp

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/stretchr/testify/require"
)

// TestPlatformMenuCarriesOnlySystemRoles pins the macOS menu as the platform's
// own chrome. Aurora's actions live in the window's Actions menu on every
// platform, so the menu bar must not grow a second copy of File/Go/Help; MyGo
// supplies Close, clipboard and window roles itself.
func TestPlatformMenuCarriesOnlySystemRoles(t *testing.T) {
	menu := NewPlatformMenu()
	require.NotNil(t, menu)

	want := []mygo.MenuRole{
		mygo.RoleAppMenu,
		mygo.RoleFileMenu,
		mygo.RoleEditMenu,
		mygo.RoleWindowMenu,
	}
	items := menu.Items()
	require.Len(t, items, len(want))
	for i, role := range want {
		require.Equal(t, role, items[i].Role, "item %d must be the %s role", i, role)
	}
}

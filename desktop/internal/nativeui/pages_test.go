package nativeui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEveryPageIsInEveryTable covers the three maps keyed by page and the two
// order slices. A page added to one table and missed in another fails quietly:
// a missing id makes the sidebar's click resolve to nothing (pageByID returns
// false and the page never opens), and a missing title renders an empty header.
func TestEveryPageIsInEveryTable(t *testing.T) {
	for _, p := range pageOrder {
		assert.Containsf(t, pageIDs, p, "page %d has no sidebar id", p)
		assert.Containsf(t, pageTitles, p, "page %d has no title", p)
		assert.NotNilf(t, pageIcons[p], "page %d has no icon", p)
	}
	// The reverse direction: a table entry for a page outside the order is a
	// page nothing can reach.
	for p := range pageIDs {
		assert.Containsf(t, pageOrder, p, "page %d has an id but is not in pageOrder", p)
	}
	for p := range pageTitles {
		assert.Containsf(t, pageOrder, p, "page %d has a title but is not in pageOrder", p)
	}
}

// TestSidebarIDsAreDistinctAndStable pins what a choice is remembered as. The
// id is what a click resolves; two pages sharing one would make the second
// unreachable, and an empty id would make the first unreachable.
func TestSidebarIDsAreDistinctAndStable(t *testing.T) {
	seen := map[string]page{}
	for p, id := range pageIDs {
		require.NotEmptyf(t, id, "page %d has an empty id", p)
		if other, dup := seen[id]; dup {
			t.Fatalf("pages %d and %d share the id %q", other, p, id)
		}
		seen[id] = p
	}
	// The lookup has to come back to the page its id came from.
	for p, id := range pageIDs {
		got, ok := pageByID(id)
		require.Truef(t, ok, "pageByID(%q) must resolve", id)
		assert.Equalf(t, p, got, "pageByID(%q) returned the wrong page", id)
	}
}

// TestModuleOrderIsPageOrderMinusSettings keeps the sidebar's two sections in
// step: the keyboard order is the module order plus Settings, and Settings
// belongs to the Application section rather than the module list.
func TestModuleOrderIsPageOrderMinusSettings(t *testing.T) {
	require.Equal(t, len(pageOrder)-1, len(moduleOrder),
		"the module section is the page order without Settings")
	assert.Equal(t, pageOrder[:len(moduleOrder)], moduleOrder, "the orders must agree where they overlap")
	assert.Equal(t, pageSettings, pageOrder[len(pageOrder)-1], "Settings is last, in the Application section")
}

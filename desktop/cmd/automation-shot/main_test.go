package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUnlabeledInteractiveFindsNamelessControls covers the check the capture
// script runs on every shot: a label or a description names a control, anything
// else leaves a screen reader announcing an empty button.
func TestUnlabeledInteractiveFindsNamelessControls(t *testing.T) {
	nodes := []treeNode{
		{Role: "button", Label: "Refresh"},
		{Role: "button", Description: "Copy the id"},
		{Role: "textField", Label: "Name"},
		{Role: "button", Value: "−"},
		{Role: "text", Value: "Not a control"},
		{Role: "treeItem"},
	}
	require.Equal(t, []string{`button "−"`, `treeItem ""`}, unlabeledInteractive(nodes))
}

// TestOnScreenDecidesByOverlap covers the check that turns a capture into
// evidence. The accessibility tree lists every element whether or not the
// viewport shows it, so "the text is in the tree" says nothing about what the
// PNG contains — three NFT stages once produced one identical image that way.
func TestOnScreenDecidesByOverlap(t *testing.T) {
	area := rect{X: 0, Y: 0, Width: 900, Height: 600}

	cases := []struct {
		name string
		box  rect
		want bool
	}{
		{"fully inside", rect{X: 20, Y: 100, Width: 860, Height: 40}, true},
		{"touching the bottom edge", rect{X: 20, Y: 560, Width: 860, Height: 40}, true},
		{"exactly the content height, starting at the top", rect{X: 0, Y: 0, Width: 900, Height: 600}, true},
		{"just below the fold", rect{X: 20, Y: 1782, Width: 200, Height: 36}, false},
		{"just above the fold", rect{X: 20, Y: -40, Width: 200, Height: 36}, false},
		{"to the right of the content", rect{X: 950, Y: 100, Width: 200, Height: 36}, false},
		{"to the left of the content", rect{X: -250, Y: 100, Width: 200, Height: 36}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, onScreen(c.box, area))
		})
	}

	// A window that starts away from the origin works the same way: the check
	// is in window coordinates, not content coordinates.
	offset := rect{X: 0, Y: 56, Width: 900, Height: 544}
	require.True(t, onScreen(rect{X: 20, Y: 100, Width: 200, Height: 36}, offset),
		"an element under the window's own top edge is visible")
	require.False(t, onScreen(rect{X: 20, Y: 10, Width: 200, Height: 36}, offset),
		"an element above the content area is not")
}

// TestWheelDirectionFollowsTheTarget covers which way the capture scrolls. The
// loop wheels down while the box sits below the view and up when it sits above,
// so a target the previous frame scrolled past is still reachable.
func TestWheelDirectionFollowsTheTarget(t *testing.T) {
	area := rect{X: 0, Y: 0, Width: 900, Height: 600}

	below := rect{X: 20, Y: 1200, Width: 200, Height: 36}
	require.False(t, onScreen(below, area))
	require.Positive(t, wheelDirection(below, area), "a target below the view scrolls down")

	above := rect{X: 20, Y: -200, Width: 200, Height: 36}
	require.False(t, onScreen(above, area))
	require.Negative(t, wheelDirection(above, area), "a target above the view scrolls up")
}

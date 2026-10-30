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

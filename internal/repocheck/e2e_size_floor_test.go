package repocheck

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTheCaptureSizeFloorMatchesTheWindow covers AURORA_E2E_SIZE against the
// window's own minimum, which lives in cmd/aurora-desktop.
//
// The script used to accept any WIDTHxHEIGHT and name the output directory
// after it. A size below the minimum is clamped when the window opens, so
// AURORA_E2E_SIZE=700x520 produced 900x600 PNGs under a directory called
// 700-520 — a reader would take them as evidence of a layout nobody saw. The
// script now refuses such a size; this check keeps the floor it refuses below
// equal to the one the window enforces.
func TestTheCaptureSizeFloorMatchesTheWindow(t *testing.T) {
	window := readRepoFile(t, "desktop", "cmd", "aurora-desktop", "main.go")
	script := readRepoFile(t, "desktop", "scripts", "e2e-window.sh")

	// The window's minimum, read from the options literal the app passes.
	minimum := regexp.MustCompile(`MinWidth:\s+(\d+),\s*\n\s*MinHeight:\s+(\d+),`)
	m := minimum.FindStringSubmatch(window)
	require.NotNil(t, m, "the window must still declare MinWidth and MinHeight in that order")

	width, err := strconv.Atoi(m[1])
	require.NoError(t, err)
	height, err := strconv.Atoi(m[2])
	require.NoError(t, err)

	floor := regexp.MustCompile(`(?m)^min_width=(\d+)$\n^min_height=(\d+)$`)
	f := floor.FindStringSubmatch(script)
	require.NotNil(t, f, "the capture script must declare the floor it enforces")

	assert.Equalf(t, width, mustAtoi(t, f[1]),
		"the script refuses sizes under min_width=%s, but the window opens at %d", f[1], width)
	assert.Equalf(t, height, mustAtoi(t, f[2]),
		"the script refuses sizes under min_height=%s, but the window opens at %d", f[2], height)
}

func mustAtoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(s)
	require.NoError(t, err)
	return n
}

// TestTheCaptureScriptResolvesItsOutputPath covers AURORA_E2E_OUT when it is
// relative. The script builds the window from $root/desktop and wrote the
// binaries to whatever "$out" said; with AURORA_E2E_OUT=dist/e2e2 the app was
// compiled to desktop/dist/e2e2 and the launch then looked for
// dist/e2e2/automation-shot, which never existed. The run failed with "No such
// file or directory" and left a stray desktop/dist behind.
//
// The fix makes the path absolute before anything reads it, so this check is
// about the ordering: the normalisation has to happen before the assignment
// that other lines depend on.
func TestTheCaptureScriptResolvesItsOutputPath(t *testing.T) {
	script := readRepoFile(t, "desktop", "scripts", "e2e-window.sh")

	outAt := strings.Index(script, `out=${AURORA_E2E_OUT:-"$root/dist/e2e"}`)
	require.NotEqual(t, -1, outAt, "the script must keep reading AURORA_E2E_OUT")

	absAt := strings.Index(script, `*) out="$PWD/$out" ;;`)
	require.NotEqual(t, -1, absAt,
		"a relative AURORA_E2E_OUT must be made absolute; the build runs from desktop/")

	buildAt := strings.Index(script, `cd "$root/desktop"`)
	require.NotEqual(t, -1, buildAt, "the build must still run from the desktop module")

	assert.Lessf(t, outAt, absAt, "the absolute path must be computed after the variable is read")
	assert.Lessf(t, absAt, buildAt,
		"the absolute path must be computed before the build, which changes directory")
}

// TestTheCaptureScrollsByWheelNotByAccessibilityAction covers how a stage
// capture reaches a body below the fold.
//
// The accessibility action (scrollIntoView) reported success and moved nothing
// for the NFT and Token write tabs: each theme's three stage captures came out
// byte-identical to the page capture, and every one of them passed. The capture
// now wheels over the content area and requires the element's box to be inside
// it, so "the page did not move" is a failure.
func TestTheCaptureScrollsByWheelNotByAccessibilityAction(t *testing.T) {
	shot := readRepoFile(t, "desktop", "cmd", "automation-shot", "main.go")

	assert.NotContains(t, shot, `"action": "scrollIntoView"`,
		"the accessibility action reports success without moving; the capture must wheel instead")
	require.Contains(t, shot, `"type": "wheel"`,
		"the capture must scroll the way a person does")
	require.Contains(t, shot, "func onScreen(",
		"the capture must decide whether the target is inside the content area")

	// The decision must be reached before the PNG is written, or a capture can
	// still photograph the wrong part of the page.
	check := strings.Index(shot, "scrollPointIntoView(win, want)")
	capture := strings.Index(shot, `client.call("capture"`)
	require.NotEqual(t, -1, check, "the scroll loop must call the wheel helper")
	require.NotEqual(t, -1, capture, "the capture call must remain")
	assert.Less(t, check, capture, "the scroll must be verified before the capture")
}

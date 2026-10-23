package api

import (
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	cssVariableRE = regexp.MustCompile(`--([a-z0-9-]+)\s*:\s*(#[0-9a-fA-F]{6})\s*;`)
	cssVarUseRE   = regexp.MustCompile(`^var\(--([a-z0-9-]+)\)$`)
	cssHexColorRE = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
)

// TestWebUITextContrast keeps the shipped palette at WCAG AA contrast for
// normal-sized text. Pastel action buttons intentionally carry the dark page
// ink instead of white, and secondary copy must remain readable on both the
// page and card backgrounds.
func TestWebUITextContrast(t *testing.T) {
	cssBytes, err := os.ReadFile(filepath.Join(realWebDir(), "css", "style.css"))
	require.NoError(t, err)
	css := string(cssBytes)

	variables := make(map[string]string)
	for _, match := range cssVariableRE.FindAllStringSubmatch(css, -1) {
		variables[match[1]] = match[2]
	}

	for _, surface := range []string{"bg-primary", "bg-card"} {
		ratio := contrastRatio(t, resolveCSSColor(t, "var(--text-secondary)", variables), resolveCSSColor(t, "var(--"+surface+")", variables))
		require.GreaterOrEqual(t, ratio, 4.5,
			"--text-secondary on --%s is %.2f:1, want at least 4.5:1", surface, ratio)
	}

	for _, class := range []string{
		"btn-lottery",
		"btn-voting",
		"btn-token",
		"btn-blockchain",
		"btn-nft",
		"btn-yes",
		"btn-no",
	} {
		block := cssRule(t, css, class)
		foreground := resolveCSSColor(t, cssDeclaration(t, block, "color"), variables)
		background := resolveCSSColor(t, cssDeclaration(t, block, "background-color"), variables)
		ratio := contrastRatio(t, foreground, background)
		require.GreaterOrEqual(t, ratio, 4.5,
			".%s text contrast is %.2f:1, want at least 4.5:1", class, ratio)
	}
}

func cssRule(t *testing.T, css, class string) string {
	t.Helper()
	re := regexp.MustCompile(`(?m)^\.` + regexp.QuoteMeta(class) + `\s*\{([^}]*)\}`)
	match := re.FindStringSubmatch(css)
	require.Len(t, match, 2, "style.css must define .%s", class)
	return match[1]
}

func cssDeclaration(t *testing.T, block, property string) string {
	t.Helper()
	re := regexp.MustCompile(`(?m)(?:^|;)\s*` + regexp.QuoteMeta(property) + `\s*:\s*([^;]+);`)
	match := re.FindStringSubmatch(block)
	require.Len(t, match, 2, "CSS rule must declare %s", property)
	return strings.TrimSpace(match[1])
}

func resolveCSSColor(t *testing.T, value string, variables map[string]string) string {
	t.Helper()
	if match := cssVarUseRE.FindStringSubmatch(value); match != nil {
		resolved, ok := variables[match[1]]
		require.True(t, ok, "CSS variable --%s must resolve to a six-digit hex color", match[1])
		return normalizeHexColor(t, resolved)
	}
	return normalizeHexColor(t, value)
}

func normalizeHexColor(t *testing.T, value string) string {
	t.Helper()
	require.Regexp(t, cssHexColorRE, value, "color must be a three- or six-digit hex value or a CSS variable")
	value = strings.ToLower(value)
	if len(value) == 4 {
		value = "#" + strings.Repeat(value[1:2], 2) + strings.Repeat(value[2:3], 2) + strings.Repeat(value[3:4], 2)
	}
	return value
}

func contrastRatio(t *testing.T, foreground, background string) float64 {
	t.Helper()
	lighter, darker := relativeLuminance(t, foreground), relativeLuminance(t, background)
	if lighter < darker {
		lighter, darker = darker, lighter
	}
	return (lighter + 0.05) / (darker + 0.05)
}

func relativeLuminance(t *testing.T, hexColor string) float64 {
	t.Helper()
	channels := []float64{}
	for offset := 1; offset < 7; offset += 2 {
		value, err := strconv.ParseInt(hexColor[offset:offset+2], 16, 64)
		require.NoError(t, err)
		channel := float64(value) / 255
		if channel <= 0.04045 {
			channel /= 12.92
		} else {
			channel = math.Pow((channel+0.055)/1.055, 2.4)
		}
		channels = append(channels, channel)
	}
	return 0.2126*channels[0] + 0.7152*channels[1] + 0.0722*channels[2]
}

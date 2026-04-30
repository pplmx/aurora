package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeaderStyle(t *testing.T) {
	style := HeaderStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestTitleStyle(t *testing.T) {
	style := TitleStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestSubtitleStyle(t *testing.T) {
	style := SubtitleStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestBodyStyle(t *testing.T) {
	style := BodyStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestCaptionStyle(t *testing.T) {
	style := CaptionStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestMenuItemStyle(t *testing.T) {
	style := MenuItemStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestMenuSelectedStyle(t *testing.T) {
	style := MenuSelectedStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestMenuActiveStyle(t *testing.T) {
	style := MenuActiveStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestMenuInactiveStyle(t *testing.T) {
	style := MenuInactiveStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestErrorStyle(t *testing.T) {
	style := ErrorStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestSuccessStyle(t *testing.T) {
	style := SuccessStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestInfoStyle(t *testing.T) {
	style := InfoStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestWarningStyle(t *testing.T) {
	style := WarningStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestBorderStyle(t *testing.T) {
	style := BorderStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestCardStyle(t *testing.T) {
	style := CardStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestButtonPrimaryStyle(t *testing.T) {
	style := ButtonPrimaryStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestButtonSecondaryStyle(t *testing.T) {
	style := ButtonSecondaryStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestInputStyle(t *testing.T) {
	style := InputStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestViewportStyle(t *testing.T) {
	style := ViewportStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestHelpTextStyle(t *testing.T) {
	style := HelpTextStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestStatusBarStyle(t *testing.T) {
	style := StatusBarStyle()
	assert.NotNil(t, style)
	assert.NotEmpty(t, style.Render("test"))
}

func TestModuleTitleStyle(t *testing.T) {
	tests := []struct {
		module string
	}{
		{"lottery"},
		{"nft"},
		{"oracle"},
		{"token"},
		{"voting"},
		{"unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.module, func(t *testing.T) {
			style := ModuleTitleStyle(tt.module)
			assert.NotNil(t, style)
			assert.NotEmpty(t, style.Render("test"))
		})
	}
}

func TestProgressBar(t *testing.T) {
	tests := []struct {
		name    string
		percent int
		width   int
	}{
		{"0%", 0, 10},
		{"50%", 50, 10},
		{"100%", 100, 10},
		{"negative clamped", -10, 10},
		{"over 100 clamped", 150, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProgressBar(tt.percent, tt.width)
			assert.NotEmpty(t, result)
		})
	}
}

func TestBadge(t *testing.T) {
	result := Badge("test", "233", "82")
	assert.NotEmpty(t, result)
}

func TestSuccessBadge(t *testing.T) {
	result := SuccessBadge("OK")
	assert.NotEmpty(t, result)
}

func TestErrorBadge(t *testing.T) {
	result := ErrorBadge("ERR")
	assert.NotEmpty(t, result)
}

func TestWarningBadge(t *testing.T) {
	result := WarningBadge("WARN")
	assert.NotEmpty(t, result)
}

func TestInfoBadge(t *testing.T) {
	result := InfoBadge("INFO")
	assert.NotEmpty(t, result)
}

func TestIcon(t *testing.T) {
	icons := []string{
		"lottery", "nft", "oracle", "token", "voting",
		"check", "cross", "warning", "info", "arrow",
		"star", "fire", "gem", "rocket", "key",
		"lock", "unlock", "plus", "minus",
	}

	for _, icon := range icons {
		t.Run(icon, func(t *testing.T) {
			result := Icon(icon)
			assert.NotEmpty(t, result)
		})
	}

	result := Icon("unknown")
	assert.Equal(t, "unknown", result)
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		expect string
	}{
		{"short string", "hi", 10, "hi"},
		{"exact length", "hello", 5, "hello"},
		{"truncate", "hello world", 8, "hello..."},
		{"empty", "", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestCenterText(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		width  int
		expect string
	}{
		{"center odd", "x", 5, "  x  "},
		{"center even", "x", 4, " x  "},
		{"full width", "hello", 5, "hello"},
		{"overflow", "hello", 3, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CenterText(tt.input, tt.width)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestSectionHeader(t *testing.T) {
	result := SectionHeader("Test Section")
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Test Section")
}

func TestKeyValue(t *testing.T) {
	result := KeyValue("key", "value")
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "key")
	assert.Contains(t, result, "value")
}

func TestDivider(t *testing.T) {
	result := Divider("-", 10)
	assert.Contains(t, result, "----------")
}

func TestBox(t *testing.T) {
	content := "Hello\nWorld"
	result := Box(content, 20)
	assert.Contains(t, result, "┌")
	assert.Contains(t, result, "┐")
	assert.Contains(t, result, "└")
	assert.Contains(t, result, "┘")
	assert.Contains(t, result, "Hello")
	assert.Contains(t, result, "World")
}

func TestCenter(t *testing.T) {
	tests := []struct {
		name    string
		width   int
		content string
	}{
		{"center 10", 10, "x"},
		{"exact width", 5, "hello"},
		{"overflow", 3, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Center(tt.width, tt.content)
			assert.NotEmpty(t, result)
		})
	}
}

func TestPadLeft(t *testing.T) {
	tests := []struct {
		name    string
		width   int
		content string
		expect  string
	}{
		{"pad 5", 5, "x", "    x"},
		{"exact", 3, "abc", "abc"},
		{"overflow", 2, "abc", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PadLeft(tt.content, tt.width)
			assert.Equal(t, tt.expect, result)
		})
	}
}

package components

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// Color values
var (
	Primary    = lipgloss.Color("86")
	Secondary  = lipgloss.Color("75")
	Accent     = lipgloss.Color("205")
	Success    = lipgloss.Color("82")
	Error      = lipgloss.Color("196")
	Warning    = lipgloss.Color("226")
	Text       = lipgloss.Color("252")
	TextMuted  = lipgloss.Color("245")
	Border     = lipgloss.Color("240")
	Background = lipgloss.Color("233")
	LotteryAcc = lipgloss.Color("212")
	NFTAccent  = lipgloss.Color("219")
	OracleAcc  = lipgloss.Color("75")
	TokenAcc   = lipgloss.Color("82")
	VotingAcc  = lipgloss.Color("205")
)

// Style factories
func HeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Primary).
		Bold(true).
		Padding(0, 1)
}

func TitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Text).
		Bold(true)
}

func SubtitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(TextMuted)
}

func BodyStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Text)
}

func CaptionStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(TextMuted)
}

func MenuItemStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Text)
}

func MenuSelectedStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Primary).
		Bold(true).
		Background(lipgloss.Color("236"))
}

func ErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Error)
}

func SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Success)
}

func InfoStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Secondary)
}

func WarningStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Warning)
}

func BorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Border)
}

func CardStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Text).
		Padding(1, 2).
		Background(lipgloss.Color("235"))
}

func ButtonPrimaryStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("233")).
		Background(Primary).
		Padding(0, 2).
		Bold(true)
}

func ButtonSecondaryStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Primary).
		Padding(0, 2)
}

// Progress bar
func ProgressBar(percent int, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := (percent * width) / 100
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return lipgloss.NewStyle().Foreground(Success).Render(bar[:filled]) +
		lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Render(bar[filled:])
}

// Badge
func Badge(text string, fg, bg string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(fg)).
		Background(lipgloss.Color(bg)).
		Padding(0, 1).
		Render(" " + text + " ")
}

func SuccessBadge(text string) string {
	return Badge(text, "233", "82")
}

func ErrorBadge(text string) string {
	return Badge(text, "233", "196")
}

func WarningBadge(text string) string {
	return Badge(text, "233", "226")
}

func InfoBadge(text string) string {
	return Badge(text, "233", "75")
}

// Icon helpers
func Icon(icon string) string {
	icons := map[string]string{
		"lottery": "🎲",
		"nft":     "🖼️ ",
		"oracle":  "🔮",
		"token":   "🪙",
		"voting":  "🗳️ ",
		"check":   "✓",
		"cross":   "✗",
		"warning": "⚠️ ",
		"info":    "ℹ️ ",
		"arrow":   "▸",
		"star":    "★",
		"fire":    "🔥",
		"gem":     "💎",
		"rocket":  "🚀",
		"key":     "🔑",
		"lock":    "🔒",
		"unlock":  "🔓",
		"plus":    "+",
		"minus":   "-",
	}
	if v, ok := icons[icon]; ok {
		return v
	}
	return icon
}

// Truncate with ellipsis
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Center text in width
func CenterText(s string, width int) string {
	padding := width - len(s)
	if padding <= 0 {
		return s
	}
	left := padding / 2
	right := padding - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// Format section header
func SectionHeader(title string) string {
	return fmt.Sprintf("\n%s %s\n%s",
		HeaderStyle().Render("▸"),
		TitleStyle().Render(title),
		BorderStyle().Render(strings.Repeat("─", 40)))
}

// Format key-value pair
func KeyValue(key, value string) string {
	return fmt.Sprintf("%s: %s",
		CaptionStyle().Render(key+":"),
		BodyStyle().Render(value))
}

// Divider
func Divider(char string, length int) string {
	return BorderStyle().Render(strings.Repeat(char, length))
}

// Box around content
func Box(content string, width int) string {
	lines := strings.Split(content, "\n")
	result := "┌" + strings.Repeat("─", width-2) + "┐\n"
	for _, line := range lines {
		padding := width - len(line) - 2
		if padding < 0 {
			padding = 0
		}
		result += "│" + line + strings.Repeat(" ", padding) + "│\n"
	}
	result += "└" + strings.Repeat("─", width-2) + "┘"
	return result
}

func ModuleTitleStyle(module string) lipgloss.Style {
	var accent color.Color
	switch module {
	case "lottery":
		accent = LotteryAcc
	case "nft":
		accent = NFTAccent
	case "oracle":
		accent = OracleAcc
	case "token":
		accent = TokenAcc
	case "voting":
		accent = VotingAcc
	default:
		accent = Primary
	}
	return lipgloss.NewStyle().
		Foreground(accent).
		Bold(true).
		Padding(0, 1)
}

func InputStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Text).
		Background(lipgloss.Color("236")).
		Padding(0, 1)
}

func ViewportStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Text).
		Background(lipgloss.Color("235")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Border)
}

func MenuActiveStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Primary).
		Bold(true).
		Background(lipgloss.Color("236"))
}

func MenuInactiveStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Text)
}

func HelpTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(TextMuted)
}

func StatusBarStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(TextMuted).
		Background(lipgloss.Color("234")).
		Padding(0, 1)
}

func Center(width int, content string) string {
	padding := width - len(content)
	if padding <= 0 {
		return content
	}
	left := padding / 2
	right := padding - left
	return strings.Repeat(" ", left) + content + strings.Repeat(" ", right)
}

func PadLeft(text string, width int) string {
	padding := width - len(text)
	if padding <= 0 {
		return text
	}
	return strings.Repeat(" ", padding) + text
}

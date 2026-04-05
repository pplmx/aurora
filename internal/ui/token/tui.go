package token

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/pplmx/aurora/internal/ui/components"
)

type model struct {
	view      string
	menuIndex int
	err       string
	success   string
}

func NewTokenApp() *model {
	return &model{
		view:      "menu",
		menuIndex: 0,
	}
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.menuIndex > 0 {
				m.menuIndex--
			}
		case "down", "j":
			if m.menuIndex < 4 {
				m.menuIndex++
			}
		case "enter":
			m.handleSelect()
		case "esc":
			m.view = "menu"
		}
	}
	return m, nil
}

func (m *model) handleSelect() {
	switch m.menuIndex {
	case 0:
		m.view = "create"
	case 1:
		m.view = "mint"
	case 2:
		m.view = "transfer"
	case 3:
		m.view = "balance"
	case 4:
		m.view = "history"
	}
}

func (m *model) View() string {
	switch m.view {
	case "menu":
		return m.menuView()
	case "create":
		return m.createView()
	case "mint":
		return m.mintView()
	case "transfer":
		return m.transferView()
	case "balance":
		return m.balanceView()
	case "history":
		return m.historyView()
	}
	return m.menuView()
}

func (m *model) menuView() string {
	s := components.HeaderStyle().Render("🪙 Aurora Token") + "\n"
	s += components.SubtitleStyle().Render("同质化代币管理系统") + "\n\n"
	s += components.Divider("─", 50) + "\n"

	menuItems := []string{
		"📦 创建代币 Create Token",
		"✨ 铸造代币 Mint Tokens",
		"💸 转账 Transfer",
		"💰 查询余额 Check Balance",
		"📜 交易历史 History",
	}

	for i, item := range menuItems {
		if i == m.menuIndex {
			s += components.MenuSelectedStyle().Render("▸ "+item) + "\n"
		} else {
			s += components.MenuItemStyle().Render("  "+item) + "\n"
		}
	}

	s += "\n" + components.CaptionStyle().Render("↑↓ 导航 | Enter 选择 | q 退出")
	return s
}

func (m *model) createView() string {
	s := components.SectionHeader("📦 创建代币 Create Token")
	s += "\n"

	card := components.CardStyle().Render(
		components.KeyValue("代币名称", "Aurora Token")+"\n"+
		components.KeyValue("代币符号", "AUR")+"\n"+
		components.KeyValue("供应量", "1,000,000")+"\n"+
		components.KeyValue("精度", "8位小数")+"\n\n"+
		components.SuccessBadge("点击确认创建"),
	)
	s += card + "\n\n"
	s += components.CaptionStyle().Render("按 Enter 确认 | ESC 返回")
	return s
}

func (m *model) mintView() string {
	s := components.SectionHeader("✨ 铸造代币 Mint Tokens")
	s += "\n"
	s += components.InfoStyle().Render("输入接收地址和数量") + "\n\n"

	card := components.CardStyle().Render(
		components.KeyValue("接收地址", "AUR123...abc")+"\n"+
		components.KeyValue("数量", "1000")+"\n\n"+
		components.KeyValue("私钥", "***隐藏***"),
	)
	s += card + "\n\n"
	s += components.CaptionStyle().Render("按 Enter 确认 | ESC 返回")
	return s
}

func (m *model) transferView() string {
	s := components.SectionHeader("💸 转账 Transfer")
	s += "\n"
	s += components.InfoStyle().Render("输入接收地址、数量和私钥") + "\n\n"

	card := components.CardStyle().Render(
		components.KeyValue("从", "你的地址")+"\n"+
		components.KeyValue("到", "目标地址")+"\n"+
		components.KeyValue("数量", "100")+"\n\n"+
		components.KeyValue("私钥", "***隐藏***"),
	)
	s += card + "\n\n"
	s += components.CaptionStyle().Render("按 Enter 确认 | ESC 返回")
	return s
}

func (m *model) balanceView() string {
	s := components.SectionHeader("💰 查询余额 Check Balance")
	s += "\n"

	balanceCard := components.CardStyle().Render(
		components.KeyValue("代币", "AUR (Aurora Token)")+"\n\n"+
		components.SuccessStyle().Render("余额: 1,000,000 AUR")+"\n\n"+
		components.KeyValue("地址", "AUR123...xyz789"),
	)
	s += balanceCard + "\n\n"
	s += components.CaptionStyle().Render("按 ESC 返回")
	return s
}

func (m *model) historyView() string {
	s := components.SectionHeader("📜 交易历史 Transaction History")
	s += "\n"

	history := components.CardStyle().Render(
		components.Icon("arrow")+" 转账\n"+
		components.KeyValue("从", "AUR123...abc")+"\n"+
		components.KeyValue("到", "XYZ789...def")+"\n"+
		components.KeyValue("数量", "100 AUR")+"\n"+
		components.CaptionStyle().Render("时间: 2024-01-15 10:30"),
	)
	s += history + "\n\n"

	history2 := components.CardStyle().Render(
		components.Icon("arrow")+" 转账\n"+
		components.KeyValue("从", "ABC456...ghi")+"\n"+
		components.KeyValue("到", "你的地址")+"\n"+
		components.KeyValue("数量", "500 AUR")+"\n"+
		components.CaptionStyle().Render("时间: 2024-01-14 15:20"),
	)
	s += history2 + "\n\n"
	s += components.CaptionStyle().Render("按 ESC 返回")
	return s
}

func SectionHeader(title string) string {
	return fmt.Sprintf("\n%s %s\n%s",
		HeaderStyle().Render("▸"),
		TitleStyle().Render(title),
		Divider("─", 50))
}

func TitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
}

func HeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true).Padding(0, 1)
}

func Divider(char string, length int) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(char + strings.Repeat(char, length))
}

func KeyValue(key, value string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(key+": ") +
		lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(value)
}

var strings = struct {
	Repeat func(string, int) string
}{
	Repeat: strings.Repeat,
}

func init() {
	strings.Repeat = strings.Repeat
}

import "strings"

func RunTokenTUI() error {
	p := tea.NewProgram(NewTokenApp())
	_, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running Token TUI: %v\n", err)
		os.Exit(1)
	}
	return nil
}
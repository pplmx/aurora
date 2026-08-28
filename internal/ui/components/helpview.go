package components

import (
	"github.com/pplmx/aurora/internal/i18n"
)

// HelpView renders a locale-aware keyboard-shortcuts screen shared by the
// TUI models. Every menu footer advertises "? for help" (help.nav), so a
// placement without a help screen leaves "?" dead; routing the content
// through i18n also keeps the help screen from leaking hardcoded CJK into an
// en-locale session (TASK-130, ISS-118).
func HelpView() string {
	s := HeaderStyle().Render("⌨ "+i18n.GetText("tui.help.title")) + "\n\n"
	s += InfoStyle().Render(i18n.GetText("tui.help.nav_header")) + "\n"
	s += "  " + i18n.GetText("tui.help.up_down") + "\n"
	s += "  " + i18n.GetText("tui.help.tab") + "\n"
	s += "  " + i18n.GetText("tui.help.enter") + "\n"
	s += "  " + i18n.GetText("tui.help.esc") + "\n"
	s += "  " + i18n.GetText("tui.help.quit") + "\n"
	s += "  " + i18n.GetText("tui.help.help") + "\n\n"
	s += HelpTextStyle().Render(i18n.GetText("tui.help.back_prompt"))
	return s
}

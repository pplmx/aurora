package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/pplmx/aurora/internal/i18n"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.SetUsageFunc(localizedUsage)
	rootCmd.SetHelpFunc(localizedHelp)
}

// localizedUsage mirrors Cobra's defaultUsageFunc so the framework-generated
// headings use Aurora's translator. Cobra's Short/Long/flag descriptions were
// already localized at declaration time, but its default usage renderer has
// hardcoded English labels.
func localizedUsage(c *cobra.Command) error {
	localizeCommandHelp(c)
	out := c.OutOrStdout()

	fmt.Fprint(out, i18n.GetText("cli.help.usage"))
	if c.Runnable() {
		fmt.Fprintf(out, "\n  %s", c.UseLine())
	}
	if c.HasAvailableSubCommands() {
		fmt.Fprintf(out, "\n  %s [command]", c.CommandPath())
	}
	if len(c.Aliases) > 0 {
		fmt.Fprintf(out, "\n\n%s\n", i18n.GetText("cli.help.aliases"))
		fmt.Fprintf(out, "  %s", c.NameAndAliases())
	}
	if c.HasExample() {
		fmt.Fprintf(out, "\n\n%s\n", i18n.GetText("cli.help.examples"))
		fmt.Fprint(out, c.Example)
	}
	if c.HasAvailableSubCommands() {
		cmds := c.Commands()
		if len(c.Groups()) == 0 {
			fmt.Fprintf(out, "\n\n%s", i18n.GetText("cli.help.available_commands"))
			writeCommandList(out, cmds, "")
		} else {
			for _, group := range c.Groups() {
				fmt.Fprintf(out, "\n\n%s", group.Title)
				writeCommandList(out, cmds, group.ID)
			}
			if !c.AllChildCommandsHaveGroup() {
				fmt.Fprintf(out, "\n\n%s", i18n.GetText("cli.help.additional_commands"))
				writeCommandList(out, cmds, "")
			}
		}
	}
	if c.HasAvailableLocalFlags() {
		fmt.Fprintf(out, "\n\n%s\n", i18n.GetText("cli.help.flags"))
		fmt.Fprint(out, trimHelpSpace(c.LocalFlags().FlagUsages()))
	}
	if c.HasAvailableInheritedFlags() {
		fmt.Fprintf(out, "\n\n%s\n", i18n.GetText("cli.help.global_flags"))
		fmt.Fprint(out, trimHelpSpace(c.InheritedFlags().FlagUsages()))
	}
	if c.HasHelpSubCommands() {
		fmt.Fprintf(out, "\n\n%s", i18n.GetText("cli.help.additional_help_topics"))
		for _, subcmd := range c.Commands() {
			if subcmd.IsAdditionalHelpTopicCommand() {
				fmt.Fprintf(out, "\n  %s %s", rpadHelp(subcmd.CommandPath(), subcmd.CommandPathPadding()), subcmd.Short)
			}
		}
	}
	if c.HasAvailableSubCommands() {
		fmt.Fprintf(out, "\n\n%s", i18n.GetTextF("cli.help.more_info", c.CommandPath()))
	}
	fmt.Fprintln(out)
	return nil
}

func writeCommandList(out io.Writer, cmds []*cobra.Command, groupID string) {
	for _, subcmd := range cmds {
		if groupID != "" && subcmd.GroupID != groupID {
			continue
		}
		if groupID == "" && subcmd.GroupID != "" {
			continue
		}
		if !subcmd.IsAvailableCommand() && subcmd.Name() != "help" {
			continue
		}
		fmt.Fprintf(out, "\n  %s %s", rpadHelp(subcmd.Name(), subcmd.NamePadding()), subcmd.Short)
	}
}

// localizedHelp mirrors Cobra's default help function while keeping the
// generated usage frame in sync with the active translator.
func localizedHelp(c *cobra.Command, _ []string) {
	localizeCommandHelp(c)
	text := c.Long
	if text == "" {
		text = c.Short
	}
	if c == c.Root() {
		text = i18n.GetText("cli.root.long")
	}
	text = trimHelpSpace(text)
	if text != "" {
		fmt.Fprintln(c.OutOrStdout(), text)
		fmt.Fprintln(c.OutOrStdout())
	}
	if c.Runnable() || c.HasSubCommands() {
		fmt.Fprint(c.OutOrStdout(), c.UsageString())
	}
}

func localizeCommandHelp(c *cobra.Command) {
	c.InitDefaultHelpFlag()
	if helpFlag := c.Flags().Lookup("help"); helpFlag != nil {
		helpFlag.Usage = i18n.GetTextF("cli.help.help_flag", c.DisplayName())
	}

	root := c.Root()
	if c == root {
		c.Short = i18n.GetText("app.name")
		c.Long = i18n.GetText("cli.root.long")
	}
	for _, subcmd := range root.Commands() {
		switch subcmd.Name() {
		case "help":
			subcmd.Short = i18n.GetText("cli.help.command")
			subcmd.Long = i18n.GetTextF("cli.help.command_long", root.DisplayName())
		case "completion":
			subcmd.Short = i18n.GetText("cli.help.completion")
			subcmd.Long = i18n.GetTextF("cli.help.completion_long", root.DisplayName())
			for _, shell := range subcmd.Commands() {
				shell.Short = i18n.GetTextF("cli.help.completion_shell", shell.Name())
			}
		case "version":
			subcmd.Short = i18n.GetText("version.short")
		}
	}
}

func trimHelpSpace(s string) string {
	return strings.TrimRight(s, " \t\r\n")
}

// rpadHelp mirrors Cobra's unexported rpad helper for the two generated lists
// that need the same column alignment as the framework renderer.
func rpadHelp(s string, padding int) string {
	if len(s) >= padding {
		return s
	}
	return s + strings.Repeat(" ", padding-len(s))
}

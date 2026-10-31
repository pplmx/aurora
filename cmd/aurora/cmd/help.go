package cmd

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/pplmx/aurora/internal/i18n"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.SetUsageFunc(localizedUsage)
	rootCmd.SetHelpFunc(localizedHelp)
	rootCmd.SetHelpCommand(newLocalizedHelpCommand())
}

// newLocalizedHelpCommand mirrors Cobra's built-in help command, whose
// unknown-topic line is hardcoded English. The completion callback and
// successful-topic behavior stay identical to Cobra's implementation.
func newLocalizedHelpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "help [command]",
		Short: i18n.GetText("cli.help.command"),
		Long:  i18n.GetTextF("cli.help.command_long", rootCmd.DisplayName()),
		ValidArgsFunction: func(c *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
			cmd, _, err := c.Root().Find(args)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			if cmd == nil {
				cmd = c.Root()
			}
			var completions []cobra.Completion
			for _, subCmd := range cmd.Commands() {
				if subCmd.IsAvailableCommand() {
					if strings.HasPrefix(subCmd.Name(), toComplete) {
						completions = append(completions, cobra.CompletionWithDesc(subCmd.Name(), subCmd.Short))
					}
				}
			}
			return completions, cobra.ShellCompDirectiveNoFileComp
		},
		Run: func(c *cobra.Command, args []string) {
			cmd, _, err := c.Root().Find(args)
			if cmd == nil || err != nil {
				c.Print(i18n.GetTextF("cli.help.unknown_topic", args))
				if usageErr := c.Root().Usage(); usageErr != nil {
					c.PrintErrln(usageErr)
				}
				return
			}
			cmd.InitDefaultHelpFlag()
			cmd.InitDefaultVersionFlag()
			if helpErr := cmd.Help(); helpErr != nil {
				c.PrintErrln(helpErr)
			}
		},
	}
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
		fmt.Fprint(out, localizeFlagDefaults(trimHelpSpace(c.LocalFlags().FlagUsages())))
	}
	if c.HasAvailableInheritedFlags() {
		fmt.Fprintf(out, "\n\n%s\n", i18n.GetText("cli.help.global_flags"))
		fmt.Fprint(out, localizeFlagDefaults(trimHelpSpace(c.InheritedFlags().FlagUsages())))
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

// localizeFlagDefaults rewrites pflag's " (default X)" annotation with the
// active translator. The flag table is the one part of the usage frame pflag
// still renders itself, so a Chinese help page ended with an English "(default
// 3)". The pattern only matches the annotation at the end of a line, never a
// word inside a description.
func localizeFlagDefaults(text string) string {
	return flagDefaultPattern.ReplaceAllStringFunc(text, func(match string) string {
		value := strings.TrimSuffix(strings.TrimPrefix(match, " (default "), ")")
		annotation := i18n.GetTextF("cli.help.default_value", value)
		if strings.HasPrefix(annotation, "（") {
			// A full-width bracket carries its own spacing; an ASCII space in
			// front of it reads as a stray gap in Chinese.
			return annotation
		}
		return " " + annotation
	})
}

var flagDefaultPattern = regexp.MustCompile(`(?m) \(default [^)]*\)$`)

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

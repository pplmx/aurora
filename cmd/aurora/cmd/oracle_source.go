// The registered feeds: adding, listing, deleting, enabling and disabling an
// oracle source. These are the commands that own a source's lifecycle, so they
// sit apart from the commands that read from one.
package cmd

import (
	"fmt"
	"strings"

	"github.com/pplmx/aurora/internal/app/oracleapp"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/spf13/cobra"
)

var sourceCmd = &cobra.Command{
	Use:   "source",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.source.cmd"),
}

var sourceAddCmd = &cobra.Command{
	Use:   "add",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.source.add"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		name, _ := cmd.Flags().GetString("name")
		url, _ := cmd.Flags().GetString("url")
		dataType, _ := cmd.Flags().GetString("type")
		interval, _ := cmd.Flags().GetInt("interval")
		method, _ := cmd.Flags().GetString("method")
		headers, _ := cmd.Flags().GetString("headers")
		path, _ := cmd.Flags().GetString("path")

		uc := oracleapp.NewAddSourceUseCase(repo)
		resp, err := uc.Execute(&oracleapp.AddSourceRequest{
			Name:     name,
			URL:      url,
			Type:     dataType,
			Method:   method,
			Headers:  headers,
			Path:     path,
			Interval: interval,
		})
		if err != nil {
			return wrapCLIError("oracle.error.register_source", err)
		}

		fmt.Printf("✅ "+i18n.GetText("oracle.source_added")+"\n", resp.Name)
		printLocalizedField("common.id", resp.ID)
		printLocalizedField("common.url", resp.URL)
		return nil
	},
}

var sourceListCmd = &cobra.Command{
	Use:   "list",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.source.list"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		limit, offset, err := pagingFlags(cmd)
		if err != nil {
			return err
		}

		search, _ := cmd.Flags().GetString("search")
		search = strings.TrimSpace(search)

		uc := oracleapp.NewListSourcesUseCase(repo)
		resp, err := uc.Execute(&oracleapp.ListSourcesRequest{Search: search, Limit: limit, Offset: offset})
		if err != nil {
			return wrapCLIError("oracle.error.list_sources", err)
		}
		if jsonRequested(cmd) {
			if resp.Sources == nil {
				resp.Sources = []*oracleapp.SourceResponse{}
			}
			return printJSON(resp.Sources)
		}

		// An offset alone asks for a page too: without it, `--offset 9` on a
		// non-empty store kept the plain heading and answered "(none)". A search
		// is a fourth answer: "Sources: 0" beside a filter claimed an empty
		// registry, when the registry may be full.
		switch {
		case limit > 0 || offset > 0:
			fmt.Printf("\n📡 %s\n", i18n.Label(i18n.GetText("oracle.sources_page"), i18n.GetTextF("common.page_with_offset", len(resp.Sources), offset)))
		case search != "":
			fmt.Printf("\n📡 %s\n", i18n.Label(i18n.GetText("oracle.sources_matches"), len(resp.Sources)))
		default:
			fmt.Printf("\n📡 %s\n", i18n.Label(i18n.GetText("oracle.sources_total"), len(resp.Sources)))
		}
		if len(resp.Sources) == 0 {
			// Four different answers: an empty store, a page past the end, a
			// page the offset skipped past, and a term that matched nothing.
			switch {
			case limit > 0 || offset > 0:
				fmt.Println("   " + i18n.GetText("common.page_empty"))
			case search != "":
				fmt.Printf("   "+i18n.GetText("oracle.source_no_match")+"\n", search)
			default:
				fmt.Println("   " + i18n.GetText("common.none"))
			}
			return nil
		}
		for _, ds := range resp.Sources {
			status := "✅ " + i18n.GetText("common.enabled")
			if !ds.Enabled {
				status = "⏳ " + i18n.GetText("common.disabled")
			}
			fmt.Printf("   - %s [%s] %s\n", ds.Name, ds.Type, status)
			printField("  "+i18n.GetText("common.id"), ds.ID)
			printField("  "+i18n.GetText("common.url"), ds.URL)
			// The path is optional: printing "Path: " with nothing after it read
			// as a missing value, so the field is left out when the source has
			// none.
			line := "     " + i18n.Label(i18n.GetText("common.method"), ds.Method)
			if ds.Path != "" {
				line += "  " + i18n.Label(i18n.GetText("common.path"), ds.Path)
			}
			line += "  " + i18n.Label(i18n.GetText("common.interval"), fmt.Sprintf("%ds", ds.Interval))
			fmt.Println(line)
			if ds.Headers != "" {
				fmt.Printf("     %s\n", i18n.Label(i18n.GetText("common.headers"), i18n.GetText("common.configured")))
			}
		}
		return nil
	},
}

var sourceDeleteCmd = &cobra.Command{
	Use:   "delete",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.source.delete"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Deleting a source is permanent; require -y/--confirm first.
		if err := requireConfirm(cmd, "cli.confirm.oracle_del"); err != nil {
			return err
		}
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		id, _ := cmd.Flags().GetString("id")

		uc := oracleapp.NewDeleteSourceUseCase(repo)
		if err := uc.Execute(id); err != nil {
			return wrapCLIError("oracle.error.delete_source", err)
		}
		fmt.Println("✅ " + i18n.GetText("oracle.source_deleted"))
		return nil
	},
}

var sourceEnableCmd = &cobra.Command{
	Use:   "enable",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.source.enable"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		id, _ := cmd.Flags().GetString("id")

		uc := oracleapp.NewEnableSourceUseCase(repo)
		if err := uc.Execute(id); err != nil {
			return wrapCLIError("oracle.error.enable_source", err)
		}
		fmt.Println("✅ " + i18n.GetText("oracle.source_enabled"))
		return nil
	},
}

var sourceDisableCmd = &cobra.Command{
	Use:   "disable",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.source.disable"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		id, _ := cmd.Flags().GetString("id")

		uc := oracleapp.NewDisableSourceUseCase(repo)
		if err := uc.Execute(id); err != nil {
			return wrapCLIError("oracle.error.disable_source", err)
		}
		fmt.Println("✅ " + i18n.GetText("oracle.source_disabled"))
		return nil
	},
}

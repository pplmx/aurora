// Presets: listing the built-in templates and adding a source from one.
package cmd

import (
	"fmt"

	"github.com/pplmx/aurora/internal/app/oracleapp"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.template.cmd"),
}

var templateListCmd = &cobra.Command{
	Use:   "list",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.template.list"),
	RunE: func(cmd *cobra.Command, args []string) error {
		templates := oracleapp.ListTemplates()
		if jsonRequested(cmd) {
			return printJSON(templates)
		}
		fmt.Printf("\n📋 %s\n", i18n.GetText("oracle.templates_heading")+i18n.Colon())
		for _, t := range templates {
			fmt.Printf("   - %s\n", t.ID)
		}
		return nil
	},
}

var templateAddCmd = &cobra.Command{
	Use:   "add",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.template.add"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		templateName, _ := cmd.Flags().GetString("template")

		template, ok := oracleapp.GetTemplate(templateName)
		if !ok {
			return newCLIError("oracle.error.template_missing", templateName)
		}

		uc := oracleapp.NewAddSourceUseCase(repo)
		resp, err := uc.Execute(&oracleapp.AddSourceRequest{
			Name:     template.Name,
			URL:      template.URL,
			Type:     template.Type,
			Method:   template.Method,
			Path:     template.Path,
			Interval: template.Interval,
		})
		if err != nil {
			return wrapCLIError("oracle.error.add_template", err)
		}

		if jsonRequested(cmd) {
			return printJSON(resp)
		}
		fmt.Printf("✅ "+i18n.GetText("oracle.template_added")+"\n", resp.Name)
		printLocalizedField("common.id", resp.ID)
		return nil
	},
}

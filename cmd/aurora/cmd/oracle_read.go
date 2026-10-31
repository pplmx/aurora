// Reading from a feed: a manual fetch, the stored values for one source, and
// the latest value. The two helpers here turn the shared selector flag into a
// validated id and clamp the query limit the way the REST handler does.
package cmd

import (
	"fmt"

	"github.com/pplmx/aurora/internal/app/oracleapp"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	infrahttp "github.com/pplmx/aurora/internal/infra/http"
	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.fetch"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		sourceID, err := resolveOracleSourceID(cmd)
		if err != nil {
			return err
		}

		fetcher := infrahttp.NewFetcher()
		defer func() { _ = fetcher.Close() }()
		uc := oracleapp.NewFetchDataUseCaseWithDeps(repo, fetcher)
		// Record CLI-fetched data on-chain, matching the API/scheduler paths
		// and the package intent; otherwise BlockHeight always printed 0.
		chain, err := blockchain.InitWithError()
		if err != nil {
			return wrapCLIError("oracle.error.init_blockchain", err)
		}
		uc.SetChain(chain)
		resp, err := uc.Execute(&oracleapp.FetchDataRequest{SourceID: sourceID})
		if err != nil {
			return wrapCLIError("oracle.error.fetch", err)
		}

		fmt.Println("✅ " + i18n.GetText("oracle.fetched"))
		printLocalizedField("common.value", resp.Value)
		fmt.Printf("   %s\n", i18n.Label(i18n.GetText("common.timestamp"), resp.Timestamp))
		fmt.Printf("   %s\n", i18n.Label(i18n.GetText("common.block_height"), resp.BlockHeight))
		return nil
	},
}

var dataCmd = &cobra.Command{
	Use:   "data",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.data.list"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		sourceID, err := resolveOracleSourceID(cmd)
		if err != nil {
			return err
		}
		limit := clampQueryLimit(cmd)

		uc := oracleapp.NewGetDataUseCase(repo)
		resp, err := uc.Execute(&oracleapp.GetDataRequest{SourceID: sourceID, Limit: limit})
		if err != nil {
			return wrapCLIError("oracle.error.get_data", err)
		}
		if jsonRequested(cmd) {
			return printJSON(resp)
		}

		fmt.Printf("\n📊 %s\n", i18n.GetText("oracle.data_heading")+i18n.Colon())
		if len(resp.Data) == 0 {
			fmt.Println("   " + i18n.GetText("common.none"))
		}
		for _, d := range resp.Data {
			fmt.Printf("   %s\n", i18n.GetTextF("oracle.data_entry", d.Timestamp, d.Value, d.BlockHeight))
		}
		return nil
	},
}

var latestCmd = &cobra.Command{
	Use:   "latest",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.latest"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		sourceID, err := resolveOracleSourceID(cmd)
		if err != nil {
			return err
		}

		uc := oracleapp.NewGetLatestDataUseCase(repo)
		resp, err := uc.Execute(&oracleapp.GetLatestDataRequest{SourceID: sourceID})
		if err != nil {
			return wrapCLIError("oracle.error.get_latest", err)
		}
		if jsonRequested(cmd) {
			return printJSON(resp)
		}
		if resp.Data == nil {
			fmt.Println(i18n.GetText("oracle.no_data"))
			return nil
		}

		fmt.Printf("\n📈 %s\n", i18n.GetText("oracle.latest_heading")+i18n.Colon())
		printLocalizedField("common.value", resp.Data.Value)
		fmt.Printf("   %s\n", i18n.Label(i18n.GetText("common.timestamp"), resp.Data.Timestamp))
		fmt.Printf("   %s\n", i18n.Label(i18n.GetText("common.block_height"), resp.Data.BlockHeight))
		return nil
	},
}

// resolveOracleSourceID returns the source selector for fetch/data/latest. An
// empty selector is a clean usage error instead of an empty lookup, so a
// mistyped invocation cannot read as "no data".
func resolveOracleSourceID(cmd *cobra.Command) (string, error) {
	if id, _ := cmd.Flags().GetString("id"); id != "" {
		return id, nil
	}
	return "", newCLIError("oracle.error.source_required")
}

// maxCLIQueryLimit mirrors the REST API handler cap (maxQueryLimit=100) so the
// CLI cannot drive an unbounded DB scan with `-l 999999999`.
const maxCLIQueryLimit = 100

// clampQueryLimit reads the data command's limit flag and clamps it into
// [1, maxCLIQueryLimit], defaulting to 10 when unset/<=0. This matches the API
// hardening added in v1.9.
func clampQueryLimit(cmd *cobra.Command) int {
	v, err := cmd.Flags().GetInt("limit")
	if err != nil || v <= 0 {
		return 10
	}
	if v > maxCLIQueryLimit {
		return maxCLIQueryLimit
	}
	return v
}

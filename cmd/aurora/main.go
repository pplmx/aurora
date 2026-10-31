// Command aurora is the command-line client for the same application layer the
// desktop window and the REST server use.
//
// Output is a table by default and JSON under --json; destructive commands
// require --confirm, and every message is translated according to the locale
// variables the shell sets.
package main

import (
	"github.com/pplmx/aurora/cmd/aurora/cmd"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/logger"
)

func main() {
	// Bootstrap diagnostics before Cobra can load [log]. The command hook
	// reinitializes from config once initConfig has populated Viper.
	logger.Init()
	i18n.DetectAndInit()

	cmd.Execute()
}

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

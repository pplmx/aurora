package main

import (
	"github.com/pplmx/aurora/cmd/aurora/cmd"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/logger"
)

var Version = "0.0.1"
var BuildTime = "unknown"

func main() {
	logger.Init()
	i18n.DetectAndInit()

	logger.Info().
		Str("version", Version).
		Str("build_time", BuildTime).
		Str("locale", i18n.GetTranslator().GetLocale()).
		Msg("Aurora starting")

	cmd.Execute()
}

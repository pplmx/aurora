package main

import (
	"github.com/pplmx/aurora/cmd/aurora/cmd"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/logger"
)

func main() {
	logger.Init()
	i18n.DetectAndInit()

	logger.Info().
		Str("version", cmd.Version).
		Str("build_time", cmd.BuildTime).
		Str("locale", i18n.GetTranslator().GetLocale()).
		Msg("Aurora starting")

	cmd.Execute()
}

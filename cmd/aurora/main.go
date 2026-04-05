package main

import (
	"github.com/pplmx/aurora/cmd/aurora/cmd"
	"github.com/pplmx/aurora/internal/logger"
)

var Version = "1.0.0"
var BuildTime = "unknown"

func main() {
	logger.Init()
	logger.Info().Str("version", Version).Str("build_time", BuildTime).Msg("Aurora starting")
	cmd.Execute()
}

package main

import (
	"github.com/pplmx/aurora/cmd/aurora/cmd"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/logger"
	"github.com/pplmx/aurora/internal/nft"
	"github.com/pplmx/aurora/internal/oracle"
	"github.com/pplmx/aurora/internal/voting"
)

var Version = "1.0.0"
var BuildTime = "unknown"

func main() {
	logger.Init()
	i18n.DetectAndInit()

	storage := voting.NewInMemoryStorage()
	voting.InitVoting(storage)

	oracleStorage := oracle.NewInMemoryStorage()
	oracle.InitOracle(oracleStorage)

	nftStorage := nft.NewNFTStorage()
	nft.SetNFTStorage(nftStorage)

	logger.Info().
		Str("version", Version).
		Str("build_time", BuildTime).
		Str("locale", i18n.GetTranslator().GetLocale()).
		Msg("Aurora starting")

	cmd.Execute()
}

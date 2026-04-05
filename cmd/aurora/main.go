/*
Copyright © 2023 Mystic
*/
package main

import (
	"github.com/pplmx/aurora/cmd/aurora/cmd"
	"github.com/pplmx/aurora/internal/logger"
)

func main() {
	logger.Init()
	cmd.Execute()
}

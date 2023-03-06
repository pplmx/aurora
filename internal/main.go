package main

import (
	"github.com/pplmx/aurora/internal/logger"
	"github.com/pplmx/aurora/test"
)

func main() {

	test.Blockchain()

	logger.Logger.Debug("hi, this is a debug message")
	logger.Logger.Info("hi, this is a info message")
	logger.Logger.Warn("hi, this is a warn message")
	logger.Logger.Error("hi, this is a error message")
	logger.Logger.Fatal("hi, this is a fatal message")

}

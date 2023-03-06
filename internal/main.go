package main

import (
	"github.com/pplmx/aurora/internal/logger"
	"github.com/pplmx/aurora/test"
	"go.uber.org/zap"
)

func main() {

	test.Blockchain()

	logger.DEBUG("hi, this is a debug message", zap.String("test", "add kv"))
	logger.INFO("hi, this is a info message")
	logger.WARN("hi, this is a warn message")
	logger.ERROR("hi, this is a error message")
	logger.FATAL("hi, this is a fatal message")

}

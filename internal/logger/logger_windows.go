//go:build windows

package logger

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	CompanyName = "Mystic"
	AppName     = "aurora"
)

var (
	CompanyDir string
	AppDir     string
	DebugLog   string
	InfoLog    string
	ErrorLog   string
)

var logger *zap.Logger

func init() {
	dir, err := os.UserConfigDir()
	if err != nil {
		fmt.Println("Failed to get windows user config dir.")
		panic(fmt.Sprintf("Failed to get windows user config dir: %v", err))
	}
	CompanyDir = dir + "\\" + CompanyName
	AppDir = CompanyDir + "\\" + AppName
	DebugLog = AppDir + "\\" + AppName + "-debug.log"
	InfoLog = AppDir + "\\" + AppName + ".log"
	ErrorLog = AppDir + "\\" + AppName + "-error.log"
	err = os.MkdirAll(AppDir, os.ModePerm)
	if err != nil {
		fmt.Printf("Failed to create dir %v.\n", AppDir)
		return
	}
	logger = newZapLogger()
}

func DEBUG(msg string, fields ...zapcore.Field) {
	logger.Debug(msg, fields...)
}

func WARN(msg string, fields ...zapcore.Field) {
	logger.Warn(msg, fields...)
}

func INFO(msg string, fields ...zapcore.Field) {
	logger.Info(msg, fields...)
}

func ERROR(msg string, fields ...zapcore.Field) {
	logger.Error(msg, fields...)
}

func FATAL(msg string, fields ...zapcore.Field) {
	logger.Fatal(msg, fields...)
}

func newZapLogger() *zap.Logger {
	// First, define our level-handling logic.
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	stdPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.InfoLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.DebugLevel
	})

	// write to the files or the stdout[stderr]
	fileDebug := getWriteSyncer(DebugLog)
	fileStd := getWriteSyncer(InfoLog)
	fileError := getWriteSyncer(ErrorLog)

	enc := zap.NewProductionEncoderConfig()
	enc.TimeKey = "time"
	enc.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
	fileEncoder := zapcore.NewJSONEncoder(enc)

	// Join the outputs, encoders, and level-handling functions into
	// zapcore.Cores, then tee the four cores together.
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, fileError, highPriority),
		zapcore.NewCore(fileEncoder, fileStd, stdPriority),
		zapcore.NewCore(fileEncoder, fileDebug, lowPriority),
	)

	// From a zapcore.Core, it's easy to construct a logger.
	//Open development mode, stack trace
	caller := zap.AddCaller()
	//Open file and line number
	development := zap.Development()
	fields := zap.Fields(zap.String("app", AppName))
	logger := zap.New(core, caller, development, fields)
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil && !errors.Is(err, syscall.ENOTTY) {
			panic(err)
		}
	}(logger)
	return logger
}

func getWriteSyncer(path string) zapcore.WriteSyncer {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}
	return file
}

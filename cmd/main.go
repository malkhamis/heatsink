package main

import (
	"log"
	"os"
	"sync"

	"go.uber.org/zap"
)

// osExit is internally used to ease unit-testing of the main function
var osExit = os.Exit

func main() {
	code := execute()
	osExit(code)
}

func execute() (exitCode int) {

	logger := newLogger()
	defer logger.Sync()

	if len(os.Args) < 2 {
		logger.Error("invalid arguments", zap.String("error", "no filepath given for json config"))
		return 64
	}
	filename := os.Args[1]

	file, err := os.Open(filename)
	if err != nil {
		logger.Error("opening the given file", zap.Error(err))
		return 66
	}

	cfg, err := newConfig(file, logger)
	if err != nil {
		logger.Error("creating heatsink config", zap.Error(err), zap.String("filename", filename))
		return 78
	}

	heatsinks, err := cfg.newHeatsinks()
	if err != nil {
		logger.Error("instantiating heatsinks", zap.Error(err), zap.String("filename", filename))
		return 78
	}

	var wg sync.WaitGroup
	for _, hs := range heatsinks {
		hs := hs
		wg.Add(1)
		go func() {
			err := hs.StartThermalControl()
			logger.Error("thermal control returned an error", zap.Error(err))
			wg.Done()
		}()
	}
	wg.Wait()

	return 1
}

// newLogger is internally used to ease unit testing
var newLogger = func() *zap.Logger {
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.OutputPaths = []string{"stdout"}
	logger := getLoggerAndPrintErrIfAny(loggerConfig.Build())
	return logger
}

// getLoggerAndPrintErrIfAny is internally used to ease unit testing
func getLoggerAndPrintErrIfAny(logger *zap.Logger, err error) *zap.Logger {
	if logger == nil {
		logger = zap.NewNop()
	}
	if err != nil {
		log.Printf("error creating logger: %v\n", err)
	}
	return logger
}

// Package logger initializes and stores a Zap logger instance in global.
package logger

import "go.uber.org/zap"

// Log is the initialized logger that can be used in other modules
var Log *zap.SugaredLogger

// Init setups the logger and stores it in global
func Init(debug bool) {
	var initLogger *zap.Logger
	if debug {
		initLogger, _ = zap.NewDevelopment()
	} else {
		initLogger, _ = zap.NewProduction()
	}
	Log = initLogger.Sugar()
}

package logger

import "go.uber.org/zap"

// Logger is the initialized logger that can be used in other modules
var Logger *zap.SugaredLogger

// Init setups the logger and stores it in global
func Init(debug bool) {
	var initLogger *zap.Logger
	if debug {
		initLogger, _ = zap.NewDevelopment()
	} else {
		initLogger, _ = zap.NewProduction()
	}
	Logger = initLogger.Sugar()
}

package logger

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// New creates a new zap logger
func New() *zap.SugaredLogger {
	viper := viper.GetViper()
	var initLogger *zap.Logger
	if viper.GetBool("debug") {
		initLogger, _ = zap.NewDevelopment()
	} else {
		initLogger, _ = zap.NewProduction()
	}
	return initLogger.Sugar()
}

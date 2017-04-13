package client

import (
	"fmt"
	"os"

	"github.com/dustinblackman/tidalwave/logger"
	"github.com/dustinblackman/tidalwave/server"
	"github.com/spf13/viper"
)

// TidalwaveClient stores the state required for log files to be processed and saved
type TidalwaveClient struct {
	Hostname string
	Channels map[string][]chan *string
	Server   *server.TidalwaveServer
}

func verifyJSON(line string) string {
	// Safety to make sure line is a full json object
	if string(line[0]) != "{" || string(line[len(line)-1]) != "}" {
		return fmt.Sprintf(`"%s"`, line)
	}
	return string(line)
}

func (tc *TidalwaveClient) writeLog(appName string, logEntry *string) {
	// TODO: Add if statement to send log to remote server if set instead of just local.
	if _, ok := tc.Channels[appName]; ok {
		for _, channel := range tc.Channels[appName] {
			channel <- logEntry
		}
	}
	if tc.Server != nil {
		tc.Server.WriteLog(appName, *logEntry)
	}
}

// AddServer adds a local server instance to client to be used for passing back logs
func (tc *TidalwaveClient) AddServer(server *server.TidalwaveServer) {
	logger.Logger.Debug("Adding server")
	tc.Server = server
}

// AddChannel adds an event channel to submit log line for an application on write.
func (tc *TidalwaveClient) AddChannel(appName string, channel chan *string) {
	if tc.Channels == nil {
		tc.Channels = map[string][]chan *string{}
	}
	tc.Channels[appName] = append(tc.Channels[appName], channel)
}

// New parses all the available flags to begin running in client mode
func New() *TidalwaveClient {
	logger.Logger.Info("Starting Client")
	viper := viper.GetViper()
	hostname, _ := os.Hostname()
	client := TidalwaveClient{Hostname: hostname}

	if len(viper.GetStringSlice("fileentry")) > 0 {
		logger.Logger.Info("Starting Log aggregator")
		client.fileAggregator(viper.GetStringSlice("fileentry"))
	}

	if len(viper.GetStringSlice("pidentry")) > 0 {
		logger.Logger.Info("Starting PID aggregator")
		client.fileAggregator(viper.GetStringSlice("pidentry"))
	}

	if viper.GetBool("docker") {
		logger.Logger.Info("Starting Docker aggregator on " + viper.GetString("docker-host"))
		client.dockerAggregator()
	}

	return &client
}

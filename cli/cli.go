package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"path"

	fsnotify "gopkg.in/fsnotify.v1"

	"github.com/dustinblackman/tidalwave/client"
	"github.com/dustinblackman/tidalwave/parser"
	"github.com/dustinblackman/tidalwave/sqlquery"
	"github.com/spf13/viper"
)

// Start creates a new instance of the CLI parser
func Start() {
	viper := viper.GetViper()
	results := parser.Query(viper.GetString("query"))

	switch res := results.(type) {
	case parser.ChannelResults:
		for line := range res.Channel {
			fmt.Println(line)
		}
	case parser.ArrayResults:
		for _, line := range *res.Results {
			fmt.Println(line)
		}
	case parser.ObjectResults:
		str, err := json.Marshal(res.Results)
		if err != nil {
			zaplog.Error("Error converting object results to JSON", err)
			return
		}
		fmt.Println(string(str))
	case parser.IntResults:
		fmt.Println(res.Results)
	}

	if viper.GetBool("tail") {
		tc := client.New()
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(err)
		}
		defer watcher.Close()

		go func() {
			for event := range watcher.Events {
				if event.Op&fsnotify.Create == fsnotify.Create {
					go tc.WatchFile(path.Base(path.Dir(event.Name)), event.Name, false)
				}
			}
		}()

		query := sqlquery.New(viper.GetString("query"))
		logChannel := make(chan *string)
		for _, appName := range query.From {
			logPaths := parser.GetLogPathsForApp(query, appName, viper.GetString("logroot"))
			tc.AddChannel(appName, logChannel)
			go tc.WatchFile(appName, logPaths[len(logPaths)-1], false)
			watcher.Add(path.Dir(logPaths[len(logPaths)-1]))
		}

		for line := range logChannel {
			fmt.Println(*line)
		}
	}
}

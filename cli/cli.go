package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dustinblackman/tidalwave/logger"
	"github.com/dustinblackman/tidalwave/parser"
	"github.com/spf13/viper"
)

// Start creates a new instance of the CLI parser
func Start() {
	viper := viper.GetViper()
	results := parser.Query(viper.GetString("query"))

	switch res := results.(type) {
	case parser.ChannelResults:
		for line := range res.Channel {
			os.Stdout.Write(line)
			os.Stdout.Write([]byte("\n"))
		}
	case parser.ArrayResults:
		for _, line := range *res.Results {
			fmt.Println(line)
		}
	case parser.ObjectResults:
		str, err := json.Marshal(res.Results)
		if err != nil {
			logger.Logger.Error("Error converting object results to JSON", err)
			return
		}
		fmt.Println(string(str))
	case parser.IntResults:
		fmt.Println(res.Results)
	}
}

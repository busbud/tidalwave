package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	"github.com/busbud/tidalwave/logger"
	"github.com/busbud/tidalwave/parser"
	"github.com/busbud/tidalwave/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	version = "HEAD"
)

func maxParallelism() int {
	maxProcs := runtime.GOMAXPROCS(0)
	numCPU := runtime.NumCPU()
	if maxProcs < numCPU {
		return maxProcs
	}
	return numCPU
}

func cliQuery() {
	viper := viper.GetViper()

	query := viper.GetString("query")
	if query == "-" {
		queryBytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			panic(err)
		}
		query = strings.TrimSpace(string(queryBytes))
	}

	results := parser.Query(query)

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
			logger.Log.Error("Error converting object results to JSON", err)
			return
		}
		fmt.Println(string(str))
	case parser.IntResults:
		fmt.Println(res.Results)
	}
}

func run(rootCmd *cobra.Command, args []string) {
	viper.AutomaticEnv()
	viper.ReadInConfig()

	// Init's global logger
	logger.Init(viper.GetBool("debug"))

	// Server and Client
	if viper.GetBool("server") {
		server.New(version)
	}

	// Cli
	if viper.GetString("query") != "" {
		cliQuery()
	}

	// If here and no query is set, then no proper flags were passed.
	if viper.GetString("query") == "" {
		rootCmd.Help()
	}
}

// New creaes a new combra command instance.
// This really only exists to make bash auto completion easier to generate.
func New() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "tidalwave",
		Example: `  tidalwave -q "SELECT * FROM myapp WHERE line.cmd = 'uptime' AND date > '2016-10-10'"`,
		Run:     run,
		Short:   "A awesomely fast JSON log parsing application queryable with SQL",
		Long: `Tidalwave is an awesomely fast command line, and server for parsing JSON logs.

Version: ` + version + `
Home: https://github.com/busbud/tidalwave`,
	}

	flags := rootCmd.PersistentFlags()
	// Shared Flags
	flags.Int("max-parallelism", maxParallelism(),
		"Set the maximum amount of threads to run when processing log files during queries. Default is the number of cores on system.")
	flags.String("logroot", "./logs", "Log root directory where log files are stored")
	flags.Bool("debug", false, "Enable debug logging")

	// Cli Flags
	flags.StringP("query", "q", "", "SQL query to execute against logs. '-' is accepted for piping in from stdin.")
	flags.Bool("skip-sort", false, "Skips sorting search queries, outputting lines as soon as they're found")

	// Server
	flags.BoolP("server", "s", false, "Start in server mode")
	flags.String("host", "0.0.0.0", "Set host IP")
	flags.String("port", "9932", "Set server PORT")

	// Load config file
	viper.SetConfigName("tidalwave")
	viper.SetEnvPrefix("tidalwave")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc")
	viper.AddConfigPath("$HOME/.tidalwave")

	// TODO: There must be a better way to load flags in to viper without rewritting them.
	for _, param := range []string{
		"max-parallelism",
		"logroot",
		"debug",
		"query",
		"skip-sort",
		"server",
		"host",
		"port"} {
		viper.BindPFlag(param, flags.Lookup(param))
	}

	return rootCmd
}

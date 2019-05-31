package parser

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/dustinblackman/moment"
	"github.com/busbud/tidalwave/logger"
	"github.com/busbud/tidalwave/sqlquery"
	"github.com/spf13/viper"
)

const (
	fileDateFormat   = "YYYY-MM-DDTHH-mm-ss"
	folderDateFormat = "YYYY-MM-DD"
)

// TidalwaveParser does stuff
type TidalwaveParser struct {
	MaxParallelism int
	LogPaths       []string
	Query          *sqlquery.QueryParams
}

// ChannelResults returns array results through a channel
type ChannelResults struct {
	Type    string
	Channel chan []byte
}

// ArrayResults does stuff
//easyjson:json
type ArrayResults struct {
	Type    string    `json:"type"`
	Results *[]string `json:"results"`
}

// IntResults does stuff
//easyjson:json
type IntResults struct {
	Type    string `json:"type"`
	Results int    `json:"results"`
}

// ObjectResults does stuff
//easyjson:json
type ObjectResults struct {
	Type    string          `json:"type"`
	Results *map[string]int `json:"results"`
}

func dateMatch(date *moment.Moment, dates []sqlquery.DateParam, dateOnly bool) bool {
	for _, dateParam := range dates {
		if !sqlquery.ProcessDate(&dateParam, *date, dateOnly) {
			return false
		}
	}

	return true
}

// GetLogPathsForApp returns all log paths matching a query for a specified app
func GetLogPathsForApp(query *sqlquery.QueryParams, appName, logRoot string) []string {
	var logPaths []string
	folderGlob, _ := filepath.Glob(path.Join(logRoot, appName+"/*/"))

	// TODO This can be optimized for single date queries.
	for _, folderPath := range folderGlob {
		folderDate := moment.New().Moment(folderDateFormat, path.Base(folderPath))
		if dateMatch(folderDate, query.Dates, true) {
			globLogs, _ := filepath.Glob(path.Join(folderPath, "/*.log"))

			for _, filename := range globLogs {
				logDate := moment.New().Moment(fileDateFormat, strings.TrimSuffix(path.Base(filename), filepath.Ext(filename)))
				if dateMatch(logDate, query.Dates, false) {
					logPaths = append(logPaths, filename)
				}
			}
		}
	}

	return logPaths
}

// GetLogPaths returns all log paths matching a query
func GetLogPaths(query *sqlquery.QueryParams, logRoot string) []string {
	var logPaths []string
	for _, appName := range query.From {
		logPaths = append(logPaths, GetLogPathsForApp(query, appName, logRoot)...)
	}

	return logPaths
}

// Query executes a given query string.
func Query(queryString string) interface{} {
	viper := viper.GetViper()

	query := sqlquery.New(queryString)
	logPaths := GetLogPaths(query, viper.GetString("logroot"))
	parser := TidalwaveParser{
		MaxParallelism: viper.GetInt("max-parallelism"),
		LogPaths:       logPaths,
		Query:          query,
	}

	logger.Log.Debugf("Log Paths: %s", logPaths)

	// TODO: Add execution time to results.
	// TODO: Need to handle nil.
	switch query.Type {
	case sqlquery.TypeCountDistinct:
		return ObjectResults{sqlquery.TypeCountDistinct, parser.CountDistinct()}
	case sqlquery.TypeDistinct:
		return ArrayResults{sqlquery.TypeDistinct, parser.Distinct()}
	case sqlquery.TypeCount:
		return IntResults{sqlquery.TypeCount, parser.Count()}
	case sqlquery.TypeSearch:
		return ChannelResults{sqlquery.TypeSearch, parser.Search()}
	default:
		return nil
	}
}

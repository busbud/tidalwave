package parser

import (
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustinblackman/tidalwave/sqlquery"
	"github.com/spf13/viper"
)

const (
	fileDateFormat   = "2006-01-02T15-04-05" // YYYY-MM-DDTHH_mm_ss
	folderDateFormat = "2006-01-02"          // YYYY-MM-DD
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
	Channel chan string
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

func dateMatch(date time.Time, dates []sqlquery.DateParam) bool {
	acceptedDatesCount := 0
	for _, dateParam := range dates {
		if (dateParam.Type == "start" && sqlquery.ProcessDate(&dateParam, date)) || (dateParam.Type == "end" && sqlquery.ProcessDate(&dateParam, date)) {
			acceptedDatesCount++
		}
	}

	return acceptedDatesCount == len(dates)
}

// GetLogPathsForApp returns all log paths matching a query for a specified app
func GetLogPathsForApp(query *sqlquery.QueryParams, appName, logRoot string) []string {
	var logPaths []string
	folderGlob, _ := filepath.Glob(path.Join(logRoot, appName+"/*/"))

	for _, folderPath := range folderGlob {
		folderDate, _ := time.Parse(folderDateFormat, path.Base(folderPath))
		if dateMatch(folderDate, query.Dates) {
			globLogs, _ := filepath.Glob(path.Join(folderPath, "/*.log"))

			for _, filename := range globLogs {
				logDate, _ := time.Parse(fileDateFormat, strings.TrimSuffix(path.Base(filename), filepath.Ext(filename)))
				if dateMatch(logDate, query.Dates) {
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
	parser := TidalwaveParser{
		MaxParallelism: viper.GetInt("max-parallelism"),
		LogPaths:       GetLogPaths(query, viper.GetString("logroot")),
		Query:          query,
	}

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

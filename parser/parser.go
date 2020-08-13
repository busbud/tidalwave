// Package parser handles parsing log files based on the SQL execution type.
package parser

import (
	"bufio"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/busbud/tidalwave/logger"
	"github.com/busbud/tidalwave/sqlquery"
	"github.com/dustinblackman/moment"
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

func readLines(logPath string, callback func(*[]byte)) error {
	var err error

	maxAttemptes := 5
	retry := 0
	for retry < maxAttemptes {
		var file *os.File
		file, err = os.Open(logPath)
		if err != nil {
			retry++
			logger.Log.Debugf("Failed to open %s after %v/%v attempts, retrying in 30 seconds. %s", logPath, retry, maxAttemptes, err.Error())
			time.Sleep(30 * time.Second)
			continue
		}

		defer file.Close() //nolint:errcheck // Don't care if there's errors.

		reader := bufio.NewReader(file)
		delim := byte('\n')

		for {
			var line []byte
			line, err = reader.ReadBytes(delim)

			if err == io.EOF {
				retry = 100
				err = nil
				break
			}

			if err != nil {
				if strings.Contains(err.Error(), "input/output error") {
					retry++
					logger.Log.Debugf("Input/output error for %s after %v/%v attempts, retrying in 30 seconds. %s", logPath, retry, maxAttemptes, err.Error())
					time.Sleep(30 * time.Second)
					break
				} else {
					return err
				}
			}

			callback(&line)
		}
	}

	return err
}

func dateMatch(date *moment.Moment, dates []sqlquery.DateParam, dateOnly bool) bool {
	for idx := range dates {
		if !sqlquery.ProcessDate(&dates[idx], *date, dateOnly) {
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

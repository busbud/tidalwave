// Package parser handles parsing log files based on the SQL execution type.
package parser

import (
	"sync"

	"github.com/busbud/tidalwave/logger"
	"github.com/busbud/tidalwave/sqlquery"
	"github.com/tidwall/gjson"
)

func distinctCountParse(query *sqlquery.QueryParams, resultsChan chan<- map[string]int, logPath string, wg *sync.WaitGroup) {
	defer wg.Done()

	results := map[string]int{}
	err := readLines(logPath, func(line *[]byte) {
		if query.ProcessLine(line) {
			res := gjson.GetBytes(*line, query.AggrPath)
			if res.Type != 0 {
				value := res.String()
				results[value]++
			}
		}
	})

	if err != nil {
		logger.Log.Fatal(err)
	}

	resultsChan <- results
}

// CountDistinct executes a COUNT(DISTINCT()) query over log results.
// SELECT COUNT(DISTINCT(line.cmd)) FROM testapp WHERE date > '2016-10-05'
func (tp *TidalwaveParser) CountDistinct() *map[string]int { //nolint:gocritic // Leave it alone.
	logsLen := len(tp.LogPaths)
	resultsChan := make(chan map[string]int, logsLen)

	var wg sync.WaitGroup
	wg.Add(logsLen + 1)

	results := []map[string]int{}
	coreLimit := make(chan bool, tp.MaxParallelism)
	go func() {
		for res := range resultsChan {
			results = append(results, res)
			<-coreLimit
			if len(results) == logsLen {
				wg.Done()
			}
		}
	}()

	for i := 0; i < logsLen; i++ {
		go distinctCountParse(tp.Query, resultsChan, tp.LogPaths[i], &wg)
		coreLimit <- true
	}

	wg.Wait()

	mergedResults := map[string]int{}
	for idx := range results {
		for key, val := range results[idx] {
			mergedResults[key] += val
		}
	}

	results = nil // Manual GC
	return &mergedResults
}

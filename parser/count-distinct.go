package parser

import (
	"log"
	"os"
	"sync"

	"github.com/dustinblackman/tidalwave/sqlquery"
	"github.com/tidwall/gjson"
)

func distinctCountParse(query *sqlquery.QueryParams, resultsChan chan<- map[string]int, logPath string, wg *sync.WaitGroup) {
	defer wg.Done()

	results := map[string]int{}
	file, err := os.Open(logPath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := createScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if query.ProcessLine(line) {
			res := gjson.Get(line, query.AggrPath)
			if res.Type != 0 {
				value := res.String()
				if _, ok := results[value]; ok {
					results[value]++
				} else {
					results[value] = 1
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	resultsChan <- results
}

// CountDistinct executes a COUNT(DISTINCT()) query over log results.
// SELECT COUNT(DISTINCT(line.cmd)) FROM testapp WHERE date > '2016-10-05'
func (tp *TidalwaveParser) CountDistinct() *map[string]int {
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
			if _, ok := mergedResults[key]; ok {
				mergedResults[key] += val
			} else {
				mergedResults[key] = val
			}
		}
	}

	results = nil // Manual GC
	return &mergedResults
}

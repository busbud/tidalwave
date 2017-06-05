package parser

import (
	"os"
	"strings"
	"sync"

	"github.com/dustinblackman/tidalwave/logger"
	"github.com/dustinblackman/tidalwave/sqlquery"
	"github.com/tidwall/gjson"
)

// LogQueryStruct contains all information about a log file, including the matching entries to the query.
type LogQueryStruct struct {
	LogPath     string
	LineNumbers [][]int
}

func searchParse(query *sqlquery.QueryParams, logStruct *LogQueryStruct, coreLimit <-chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	logger.Logger.Debugf("Processing: %s", logStruct.LogPath)
	file, err := os.Open(logStruct.LogPath)
	if err != nil {
		logger.Logger.Fatal(err)
	}
	defer file.Close()

	lineNumber := -1
	lastLineNumber := -1
	scanner := createScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		lineNumber++

		if query.ProcessLine(&line) {
			if lineNumber == (lastLineNumber+1) && lineNumber != 0 {
				logStruct.LineNumbers[len(logStruct.LineNumbers)-1][1] = lineNumber
			} else {
				logStruct.LineNumbers = append(logStruct.LineNumbers, []int{lineNumber, lineNumber})
			}
			lastLineNumber = lineNumber
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Logger.Fatal(err)
	}

	<-coreLimit
}

func searchSubmit(query *sqlquery.QueryParams, logStruct *LogQueryStruct, submitChannel chan<- string) {
	file, err := os.Open(logStruct.LogPath)
	if err != nil {
		logger.Logger.Fatal(err)
	}
	defer file.Close()

	scanner := createScanner(file)
	lineNumber := -1
	// TODO: Handle scanner errors
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++
		acceptLine := false
		// TODO: Can this be better? Faster?
		for _, lineRange := range logStruct.LineNumbers {
			if lineNumber >= lineRange[0] && lineNumber <= lineRange[1] {
				acceptLine = true
				break
			}
		}
		if !acceptLine {
			continue
		}

		// If there were select statements, join those in to a smaller JSON object.
		if len(query.Selects) > 0 {
			selectedEntries := []string{}
			for _, entry := range query.Selects {
				res := gjson.Get(line, entry.KeyPath)
				if res.Type == gjson.Number || res.Type == gjson.JSON {
					selectedEntries = append(selectedEntries, `"`+entry.KeyPath+`":`+res.String())
				} else if res.Type == gjson.True {
					selectedEntries = append(selectedEntries, `"`+entry.KeyPath+`":true`)
				} else if res.Type == gjson.False {
					selectedEntries = append(selectedEntries, `"`+entry.KeyPath+`":false`)
				} else if res.Type == gjson.Null {
					selectedEntries = append(selectedEntries, `"`+entry.KeyPath+`":null`)
				} else {
					selectedEntries = append(selectedEntries, `"`+entry.KeyPath+`":"`+res.String()+`"`)
				}
			}

			submitChannel <- "{" + strings.Join(selectedEntries, ",") + "}"
		} else {
			// Return entire log line.
			submitChannel <- line
		}
	}
}

// Search executes a normal match query over log results.
// SELECT * FROM testapp WHERE date > '2016-10-05'
func (tp *TidalwaveParser) Search() chan string {
	var wg sync.WaitGroup
	logsLen := len(tp.LogPaths)
	wg.Add(logsLen)

	submitChannel := make(chan string)
	go func() {
		coreLimit := make(chan bool, tp.MaxParallelism)
		logs := make([]LogQueryStruct, logsLen)
		for idx, logPath := range tp.LogPaths {
			logs[idx] = LogQueryStruct{LogPath: logPath}
			go searchParse(tp.Query, &logs[idx], coreLimit, &wg)
			coreLimit <- true
		}

		wg.Wait()

		for idx := range logs {
			if len(logs[idx].LineNumbers) > 0 {
				searchSubmit(tp.Query, &logs[idx], submitChannel)
			}
		}

		close(submitChannel)
	}()

	return submitChannel
}

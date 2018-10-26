package parser

import (
	"bufio"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/dustinblackman/gjson"
	"github.com/dustinblackman/tidalwave/logger"
	"github.com/dustinblackman/tidalwave/sqlquery"
	"github.com/spf13/viper"
)

// LogQueryStruct contains all information about a log file, including the matching entries to the query.
type LogQueryStruct struct {
	LogPath     string
	LineNumbers [][]int
}

func formatLine(query *sqlquery.QueryParams, line []byte) []byte {
	// If there were select statements, join those in to a smaller JSON object.
	if len(query.Selects) > 0 {
		selectedEntries := []string{}
		for idx, res := range gjson.GetManyBytes(line, query.Selects...) {
			keyName := ""
			for _, queryParam := range query.Queries {
				if queryParam.KeyPath == query.Selects[idx] {
					keyName = queryParam.KeyName
					break
				}
			}

			if res.Type == gjson.Number || res.Type == gjson.JSON {
				selectedEntries = append(selectedEntries, `"`+keyName+`":`+res.String())
			} else if res.Type == gjson.True {
				selectedEntries = append(selectedEntries, `"`+keyName+`":true`)
			} else if res.Type == gjson.False {
				selectedEntries = append(selectedEntries, `"`+keyName+`":false`)
			} else if res.Type == gjson.Null {
				selectedEntries = append(selectedEntries, `"`+keyName+`":null`)
			} else {
				selectedEntries = append(selectedEntries, `"`+keyName+`":"`+res.String()+`"`)
			}
		}

		return []byte("{" + strings.Join(selectedEntries, ",") + "}")
	}

	return line
}

func searchParse(query *sqlquery.QueryParams, logStruct *LogQueryStruct, coreLimit <-chan bool, submitChannel chan<- []byte, wg *sync.WaitGroup) {
	defer wg.Done()

	logger.Log.Debugf("Processing: %s", logStruct.LogPath)
	file, err := os.Open(logStruct.LogPath)
	if err != nil {
		logger.Log.Fatal(err)
	}
	defer file.Close()

	lineNumber := -1
	lastLineNumber := -1
	reader := bufio.NewReader(file)
	delim := byte('\n')

	for {
		line, err := reader.ReadBytes(delim)
		lineNumber++
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Log.Fatal(err)
		}

		if query.ProcessLine(&line) {
			if viper.GetBool("skip-sort") {
				submitChannel <- formatLine(query, line)
				continue
			}

			if lineNumber == (lastLineNumber+1) && lineNumber != 0 {
				logStruct.LineNumbers[len(logStruct.LineNumbers)-1][1] = lineNumber
			} else {
				logStruct.LineNumbers = append(logStruct.LineNumbers, []int{lineNumber, lineNumber})
			}
			lastLineNumber = lineNumber
		}
	}

	<-coreLimit
}

func searchSubmit(query *sqlquery.QueryParams, logStruct *LogQueryStruct, submitChannel chan<- []byte) {
	file, err := os.Open(logStruct.LogPath)
	if err != nil {
		logger.Log.Fatal(err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	delim := byte('\n')
	lineNumber := -1
	// TODO: Handle scanner errors
	for {
		line, err := reader.ReadBytes(delim)
		lineNumber++
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Log.Fatal(err)
		}

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

		submitChannel <- formatLine(query, line)
	}
}

// Search executes a normal match query over log results.
// SELECT * FROM testapp WHERE date > '2016-10-05'
func (tp *TidalwaveParser) Search() chan []byte {
	var wg sync.WaitGroup
	logsLen := len(tp.LogPaths)
	wg.Add(logsLen)

	submitChannel := make(chan []byte, 10000)
	go func() {
		coreLimit := make(chan bool, tp.MaxParallelism)
		logs := make([]LogQueryStruct, logsLen)
		for idx, logPath := range tp.LogPaths {
			logs[idx] = LogQueryStruct{LogPath: logPath}
			go searchParse(tp.Query, &logs[idx], coreLimit, submitChannel, &wg)
			coreLimit <- true
		}

		wg.Wait()

		if !viper.GetBool("skip-sort") {
			for idx := range logs {
				if len(logs[idx].LineNumbers) > 0 {
					searchSubmit(tp.Query, &logs[idx], submitChannel)
				}
			}
		}

		close(submitChannel)
	}()

	return submitChannel
}

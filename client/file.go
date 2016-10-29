package client

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/hpcloud/tail"
)

// WatchFile starts tailing a log file and submits the lines back to the client.
func (tc *TidalwaveClient) WatchFile(appName, filePath string, shouldFormatLog bool) {
	logrus := logrus.WithFields(logrus.Fields{
		"module": "client",
		"client": "file",
	})
	logrus.Debug(fmt.Sprintf("Tailing for %s: %s", appName, filePath))
	// viper := viper.GetViper()
	// syslogEnabled := viper.GetBool("syslog")

	t, err := tail.TailFile(filePath, tail.Config{
		Follow:    true,
		Location:  &tail.SeekInfo{Offset: 0, Whence: os.SEEK_END},
		MustExist: true,
		ReOpen:    false,
	})
	if err != nil {
		logrus.Fatal(err)
		return
	}

	for line := range t.Lines {
		if line.Err != nil {
			logrus.Fatal(line.Err) // TODO Change
			continue
		}

		if len(line.Text) == 0 {
			continue
		}

		// TODO: Check if this works...
		// if syslogEnabled {
		// 	splitLine := strings.Split(line.Text, "{")
		// 	line.Text = strings.Join(append(splitLine[:0], splitLine[1:]...), "{")
		// }

		line.Text = verifyJSON(line.Text)
		if shouldFormatLog {
			formattedLog := fmt.Sprintf(`{"time":%s,"hostname":"%s","line":%s}`, time.Now().UTC().Format(time.RFC3339), tc.Hostname, line.Text)
			tc.writeLog(appName, &formattedLog)
		} else {
			tc.writeLog(appName, &line.Text)
		}
	}
}

func (tc *TidalwaveClient) fileAggregator(logEntries []string) {
	for _, entry := range logEntries {
		entry := strings.Split(entry, "=")
		go tc.WatchFile(entry[0], entry[1], true)
	}
}

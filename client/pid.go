package client

import (
	"io/ioutil"
	"runtime"
	"strings"

	"github.com/Sirupsen/logrus"

	fsnotify "gopkg.in/fsnotify.v1"
)

func (tc *TidalwaveClient) readPidFile(appName, pidPath string) {
	pidByte, err := ioutil.ReadFile(pidPath)
	if err != nil {
		logrus.Fatal(err)
	}
	pid := strings.Split(string(pidByte), "\n")[0]

	// TODO: Need to verify these will end when the process exits
	go tc.WatchFile(appName, "/proc/"+pid+"/fd/1", true)
	go tc.WatchFile(appName, "/proc/"+pid+"/fd/2", true)
}

func (tc *TidalwaveClient) pidAggregator(files []string) {
	logrus := logrus.WithFields(logrus.Fields{
		"module": "client",
		"client": "pid",
	})
	if runtime.GOOS != "linux" {
		logrus.Fatal("Only linux is supported for grabbing stdout/stderr from a running application")
	}

	for _, entry := range files {
		entry := strings.Split(entry, "=")
		tc.readPidFile(entry[0], entry[1])

		go func() {
			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				logrus.Fatal(err)
			}

			for event := range watcher.Events {
				if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write {
					tc.readPidFile(entry[0], entry[1])
				}
			}

			defer watcher.Close()
		}()
	}
}

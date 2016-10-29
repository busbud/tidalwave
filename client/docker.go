package client

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	"github.com/spf13/viper"
)

// Borrowed from https://github.com/jwilder/docker-gen/blob/master/context.go
func getCurrentContainerID() string {
	file, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return ""
	}

	reader := bufio.NewReader(file)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	regex := "/docker[/-]([[:alnum:]]{64})(\\.scope)?$"
	re := regexp.MustCompilePOSIX(regex)

	for scanner.Scan() {
		_, lines, err := bufio.ScanLines([]byte(scanner.Text()), true)
		if err == nil {
			if re.MatchString(string(lines)) {
				submatches := re.FindStringSubmatch(string(lines))
				containerID := submatches[1]

				return containerID
			}
		}
	}
	return ""
}

func (tc *TidalwaveClient) pipeContainer(client *docker.Client, container *docker.Container) {
	logrus := logrus.WithFields(logrus.Fields{
		"module": "client",
		"client": "docker",
	})
	containerName := container.Name[1:len(container.Name)]
	appName := "docker"
	for _, env := range container.Config.Env {
		if strings.Contains(env, "TIDALWAVE_NAME") {
			appName = strings.Split(env, "=")[1]
			break
		}
	}

	logrus.Debug("Listening for events for " + containerName)
	r, w := io.Pipe()
	go func() {
		client.Logs(docker.LogsOptions{
			Container:    container.ID,
			OutputStream: w,
			ErrorStream:  w,
			Stdout:       true,
			Stderr:       true,
			Follow:       true,
			Tail:         "0",
		})
		w.Close()
		logrus.Debug("No longer listening for events for " + containerName)
	}()

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		formattedLog := fmt.Sprintf(
			`{"time":%s,"hostname":"%s","container":"%s","image":"%s","line":%s}`,
			time.Now().UTC().Format(time.RFC3339),
			container.Config.Hostname,
			containerName,
			container.Config.Image,
			verifyJSON(line))
		tc.writeLog(appName, &formattedLog)
	}
	if err := scanner.Err(); err != nil {
		logrus.Fatal(err)
	}
}

func (tc *TidalwaveClient) dockerAggregator() {
	viper := viper.GetViper()
	client, err := docker.NewClient(viper.GetString("docker-host"))
	if err != nil {
		logrus.Fatal(err)
		return
	}

	containers, err := client.ListContainers(docker.ListContainersOptions{All: false, Size: false})
	if err != nil {
		logrus.Fatal(err)
		return
	}

	selfID := getCurrentContainerID()
	for _, container := range containers {
		if container.ID == selfID {
			continue
		}

		containerDetails, err := client.InspectContainer(container.ID)
		if err != nil {
			logrus.Fatal(err)
		}
		go tc.pipeContainer(client, containerDetails)
	}

	go func() {
		dockerEvents := make(chan *docker.APIEvents, 100)
		client.AddEventListener(dockerEvents)
		for event := range dockerEvents {
			if event.Status == "start" {
				containerDetails, err := client.InspectContainer(event.ID)
				if err != nil {
					logrus.Fatal(err)
				}
				go tc.pipeContainer(client, containerDetails)
			}
		}
	}()
}

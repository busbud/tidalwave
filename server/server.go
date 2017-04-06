package server

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/dustinblackman/tidalwave/parser"
	"github.com/labstack/echo"
	fastengine "github.com/labstack/echo/engine/fasthttp"
	"github.com/labstack/echo/middleware"
	"github.com/spf13/viper"
)

const (
	fileDateFormat = "2006-01-02T15-04-05" // YYYY-MM-DDTHH-mm-ss
)

// TidalwaveServer stores the state required for the server to operate.
type TidalwaveServer struct {
	LogRoot        string
	SocketsManager *SocketsManager
}

// WriteLog saves log entries to disk
func (ts *TidalwaveServer) WriteLog(appName, logEntry string) {
	ts.SocketsManager.NewLinesChannel <- NewLogLine{appName, logEntry}

	logDate := time.Now().Format(fileDateFormat)
	logPath := path.Join(ts.LogRoot, appName, time.Now().Format("2006-01-02"))
	logFile := path.Join(logPath, fmt.Sprintf("%s_00_00.log", logDate))

	if _, err := os.Stat(logPath); err != nil {
		os.MkdirAll(logPath, 0777)
	}

	var fileHandle *os.File
	if _, err := os.Stat(logFile); err != nil {
		fileHandle, _ = os.Create(logFile)
	} else {
		fileHandle, _ = os.OpenFile(logFile, os.O_RDWR|os.O_APPEND, 0666)
	}

	writer := bufio.NewWriter(fileHandle)
	defer fileHandle.Close()

	fmt.Fprintln(writer, logEntry)
	writer.Flush()
}

func jsonError(ctx echo.Context, err error) {
	ctx.JSON(500, map[string]string{"error": err.Error()})
}

// New creates and starts the API server
func New(version string) *TidalwaveServer {
	logrus := logrus.WithFields(logrus.Fields{"module": "server"})
	logrus.Info("Starting Server")
	viper := viper.GetViper()
	server := TidalwaveServer{viper.GetString("logroot"), NewSocketsManager()}

	app := echo.New()
	app.Use(middleware.Gzip())
	app.Use(middleware.CORS())
	app.Use(middleware.Logger())
	app.Use(middleware.Recover())

	app.Get("/socket", server.SocketsManager.StartConnection)
	app.Get("/query", func(ctx echo.Context) error {
		queryString := ctx.QueryParam("q")
		if len(queryString) < 6 {
			ctx.JSON(400, map[string]string{"error": "Query length needs to be greater then 6"})
			return nil
		}

		start := time.Now()
		queryResults := parser.Query(queryString)

		switch results := queryResults.(type) {
		case parser.ChannelResults:
			r, w := io.Pipe()
			go ctx.Stream(200, "application/json", r)
			w.Write([]byte(`{"type":"` + results.Type + `","results":[`))

			first := true
			for line := range results.Channel {
				if first {
					w.Write([]byte(line))
					first = false
				} else {
					w.Write([]byte("," + line)) // TODO This breaks sometimes and is missing a comma (wat?)
				}
			}
			w.Write([]byte("]}"))
			w.Close()
		case parser.ArrayResults:
			if bytes, err := results.MarshalJSON(); err != nil {
				jsonError(ctx, err)
			} else {
				ctx.JSONBlob(200, bytes)
			}
		case parser.ObjectResults:
			if bytes, err := results.MarshalJSON(); err != nil {
				jsonError(ctx, err)
			} else {
				ctx.JSONBlob(200, bytes)
			}
		case parser.IntResults:
			if bytes, err := results.MarshalJSON(); err != nil {
				jsonError(ctx, err)
			} else {
				ctx.JSONBlob(200, bytes)
			}
		default:
			ctx.JSON(400, map[string]string{"error": "Query type not supported"})
		}

		elapsed := time.Since(start)
		logrus.Debug("Execution time: %s\n", elapsed)
		return nil
	})

	go app.Run(fastengine.New(":" + viper.GetString("port")))

	return &server
}

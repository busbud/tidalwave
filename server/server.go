package server

import (
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dustinblackman/tidalwave/logger"
	"github.com/dustinblackman/tidalwave/parser"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/spf13/viper"
)

const (
	fileDateFormat = "2006-01-02T15-04-05" // YYYY-MM-DDTHH-mm-ss
)

func jsonError(ctx echo.Context, err error) {
	ctx.JSON(500, map[string]string{"error": err.Error()})
}

// New creates and starts the API server
func New(version string) {
	logger.Log.Info("Starting Server")
	viper := viper.GetViper()

	app := echo.New()
	app.Use(middleware.Gzip())
	app.Use(middleware.CORS())
	app.Use(middleware.Logger())
	app.Use(middleware.Recover())

	app.GET("/query", func(ctx echo.Context) error {
		queryString := ctx.QueryParam("q")
		if len(queryString) < 6 {
			// TODO Silly error.
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
					w.Write(line)
					first = false
				} else {
					w.Write([]byte(",")) // TODO This breaks sometimes and is missing a comma (wat?)
					w.Write(line)
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
		logger.Log.Debug("Execution time: %s\n", elapsed)
		return nil
	})

	go app.Start(":" + viper.GetString("port"))

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for range c {
		logger.Log.Info("Exit signal received, closing...")
		app.Close()
		os.Exit(0)
	}
}

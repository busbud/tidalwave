// Package server is a proof of concept to enable an HTTP interface to Tidalwave.
package server

import (
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/busbud/tidalwave/logger"
	"github.com/busbud/tidalwave/parser"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
)

func jsonError(ctx echo.Context, err error) error {
	return ctx.JSON(500, map[string]string{"error": err.Error()})
}

// New creates and starts the API server
func New(version string) {
	logger.Log.Info("Starting Server")

	app := echo.New()
	app.HideBanner = true
	app.Use(middleware.Gzip())
	app.Use(middleware.CORS())
	app.Use(middleware.Logger())
	app.Use(middleware.Recover())

	app.GET("/", func(ctx echo.Context) error {
		return ctx.JSON(200, map[string]string{"status": "up"})
	})

	app.GET("/query", func(ctx echo.Context) error {
		queryString := ctx.QueryParam("q")
		logger.Log.Debug(map[string]string{"query": queryString})
		if len(queryString) < 6 {
			// TODO Silly error.
			return ctx.JSON(400, map[string]string{"error": "Query length needs to be greater then 6"})
		}

		start := time.Now()
		defer func() {
			elapsed := time.Since(start)
			logger.Log.Debug("Execution time: %s\n", elapsed)
		}()

		queryResults := parser.Query(queryString)

		switch results := queryResults.(type) {
		case parser.ChannelResults:
			r, w := io.Pipe()
			go ctx.Stream(200, "application/json", r) //nolint:errcheck // Don't care if there's errors.
			_, err := w.Write([]byte(`{"type":"` + results.Type + `","results":[`))
			if err != nil {
				logger.Log.Debug(err)
			}

			first := true
			for line := range results.Channel {
				if first {
					_, err = w.Write(line)
					if err != nil {
						logger.Log.Warn(err)
					}

					first = false
				} else {
					_, err = w.Write([]byte(",")) // TODO This breaks sometimes and is missing a comma (wat?)
					if err != nil {
						logger.Log.Warn(err)
					}

					_, err = w.Write(line)
					if err != nil {
						logger.Log.Warn(err)
					}
				}
			}

			_, err = w.Write([]byte("]}"))
			if err != nil {
				logger.Log.Warn(err)
			}

			err = w.Close()
			if err != nil {
				logger.Log.Warn(err)
			}
		case parser.ArrayResults:
			bytes, err := results.MarshalJSON()
			if err != nil {
				return jsonError(ctx, err)
			}
			return ctx.JSONBlob(200, bytes)
		case parser.ObjectResults:
			bytes, err := results.MarshalJSON()
			if err != nil {
				return jsonError(ctx, err)
			}
			return ctx.JSONBlob(200, bytes)
		case parser.IntResults:
			bytes, err := results.MarshalJSON()
			if err != nil {
				return jsonError(ctx, err)
			}
			return ctx.JSONBlob(200, bytes)
		default:
			return ctx.JSON(400, map[string]string{"error": "Query type not supported"})
		}

		return nil
	})

	app.GET("/query-for-lines", func(ctx echo.Context) error {
		queryString := ctx.QueryParam("q")
		logger.Log.Debug(map[string]string{"query": queryString})
		if len(queryString) < 6 {
			// TODO Silly error.
			return ctx.JSON(400, map[string]string{"error": "Query length needs to be greater then 6"})
		}

		start := time.Now()
		defer func() {
			elapsed := time.Since(start)
			logger.Log.Debug("Execution time: %s\n", elapsed)
		}()

		queryResults := parser.Query(queryString)

		switch results := queryResults.(type) {
		case parser.ChannelResults:
			r, w := io.Pipe()
			go ctx.Stream(200, "application/json-seq", r) //nolint:errcheck // Don't care if there's errors.
			for line := range results.Channel {
				_, err := w.Write([]byte("\n"))
				if err != nil {
					logger.Log.Warn(err)
				}

				_, err = w.Write(line)
				if err != nil {
					logger.Log.Warn(err)
				}
			}
			err := w.Close()
			if err != nil {
				logger.Log.Warn(err)
			}
		case parser.ArrayResults:
			return ctx.JSON(400, map[string]string{"error": "Array results not supportred on /query-by-line. Use /query instead."})
		case parser.ObjectResults:
			return ctx.JSON(400, map[string]string{"error": "Object results not supportred on /query-by-line. Use /query instead."})
		case parser.IntResults:
			return ctx.JSON(400, map[string]string{"error": "Integer results not supportred on /query-by-line. Use /query instead."})
		default:
			return ctx.JSON(400, map[string]string{"error": "Query type not supported"})
		}

		return nil
	})

	go app.Start(":" + viper.GetString("port")) //nolint:errcheck // Don't care if there's errors.

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for range c {
		logger.Log.Info("Exit signal received, closing...")
		err := app.Close()
		if err != nil {
			logger.Log.Debug(err)
		}
		os.Exit(0)
	}
}

package main

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	flickrdownconfig "github.com/jpg0/flickrdown/config"
	"github.com/juju/errors"
	"github.com/rickb777/date"
	"github.com/urfave/cli"
	"os"
	"strings"
)

const input_layout string = "2006-01-02"

func main() {
	app := cli.NewApp()
	app.Name = "flickrdown"
	app.Usage = "Download photos from Flickr"
	app.Version = "1.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config",
			Usage: "File path to configuration file",
		},
		cli.StringFlag{
			Name:  "loglevel",
			Usage: "Logging level",
			Value: "info",
		},
		cli.StringFlag{
			Name:  "startdate",
			Usage: "Date from which to begin processing",
		},
		cli.StringFlag{
			Name:  "enddate",
			Usage: "Date at which to complete processing",
		},
	}
	app.Action = verbose(watch)
	err := app.Run(os.Args)

	if err != nil {
		fmt.Errorf("Error occurred: %v", err)
		os.Exit(-1)
	}
}

func verbose(next func(*cli.Context) error) func(*cli.Context) error {
	return func(c *cli.Context) error {
		err := next(c)

		if err != nil {
			fmt.Println(errors.ErrorStack(err))
		}

		return err
	}
}

func initLogging(level string) error {
	switch strings.ToLower(level) {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "fatal":
		logrus.SetLevel(logrus.FatalLevel)
	default:
		return errors.Errorf("Unknown logging level: %v", level)
	}

	return nil
}

func watch(c *cli.Context) error {

	err := initLogging(c.String("loglevel"))

	if err != nil {
		return errors.Trace(err)
	}

	configfile := c.String("config")

	if configfile == "" {
		fmt.Println("No config file specified")
		os.Exit(-1)
	}

	config, err := flickrdownconfig.Load(configfile)

	if err != nil {
		return errors.Trace(err)
	}

	var startDate date.Date

	if c.String("startdate") != "" {
		startDate, err = date.Parse(input_layout, c.String("startdate"))
		if err != nil {
			return errors.Trace(err)
		}
	}

	endDate := startDate.Add(1)

	if c.String("enddate") != "" {
		endDate, err = date.Parse(input_layout, c.String("enddate"))
		if err != nil {
			return errors.Trace(err)
		}
	}


	return BeginBatchDownload(startDate, endDate, config)
}

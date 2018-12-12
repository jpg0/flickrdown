package main

import (
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/jpg0/flickrdown/flickraccess"
	"github.com/juju/errors"
	"os"
	"time"
)

import "github.com/jpg0/flickrdown/config"

var minstart = time.Date(2000, 0, 0, 0, 0, 0, 0, time.UTC)

func BeginBatchDownload(startAt time.Time, endAt time.Time, config *config.Config) error {

	if !isDate(startAt) {
		return errors.New("startDate is not a date (has a time component)")
	}

	if startAt.Before(minstart) {
		return errors.Errorf("Cannot begin batch earlier than %v", minstart)
	}

	if !isDate(endAt) {
		return errors.New("endDate is not a date (has a time component)")
	}

	state, err := loadState(config.StateFile)

	if err != nil {
		return errors.Annotate(err, "Failed to load state")
	}

	realStart := startAt
//	saveState := false

	if realStart.Equal(time.Time{}) {
		realStart = state.startAt
//		saveState = true
	}

	ctx, err := buildContext(config)

	if err != nil {
		return errors.Annotate(err, "Failed to build context")
	}

	return DownloadForDay(startAt, ctx)
}

func buildContext(config *config.Config) (*DownloadingContext, error) {

	client, err := flickraccess.NewDownloadClient(config)

	if err != nil {
		return nil, errors.Annotatef(err, "Failed to create flickr client")
	}

	return &DownloadingContext{
		flickrclient: client,
	}, nil
}

func DownloadForDay(day time.Time, ctx *DownloadingContext) error {

	//first query for all photos that day
	batch := ctx.flickrclient.Search(day, day.AddDate(0, 0, 1))

	for {

		photo, err := batch.NextPhoto()

		if err != nil {
			return errors.Annotate(err, "Failed to load photo")
		}

		processPhoto(photo)

	//	if photo == nil {
			break
	//	}
	}

	return nil
}

func processPhoto(photo *flickraccess.RemotePhoto) {
	meta, err := photo.GetMeta()

	if err != nil {
		logrus.Errorf("Failed to get metadata: %v", err)
	}

	asJson, err := json.Marshal(meta)

	if err != nil {
		logrus.Errorf("Failed to marshal metadata: %v", err)
	}

	fmt.Println(string(asJson))
}

func isDate(time time.Time) bool {
	h, m, s := time.Clock()
	return h == 0 && m == 0 && s == 0
}

type DownloadingContext struct {
	flickrclient *flickraccess.FlickrDownloadClient
}

type SavedState struct {
	startAt time.Time
}


func loadState(filename string) (*SavedState, error){

	rv := new(SavedState)
	stat, err := os.Stat(filename)

	if os.IsNotExist(err) {
		logrus.Info("Creating new state file")
		_, err = os.Create(filename)
		if err != nil {
			errors.Annotatef(err, "Failed to create new state file: %v", filename)
		}

		return rv, nil

	}

	if stat.Size() == 0 {
		logrus.Warn("Empty state file found, assuming no initial state")
		return rv, nil
	}

	return rv, config.LoadTo(filename, rv)
}
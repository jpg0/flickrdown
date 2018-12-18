package main

import (
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/jpg0/flickrdown/flickraccess"
	"github.com/juju/errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"github.com/rickb777/date"
)

import "github.com/jpg0/flickrdown/config"

var minstart = date.New(2000, 0, 0)
const flicrkDateFormat = "2006-01-02 15:04:05"

func BeginBatchDownload(startAt date.Date, endAt date.Date, config *config.Config) error {

	logrus.Debugf("Beginning batch download")

	if startAt.Before(minstart) {
		return errors.Errorf("Cannot begin batch earlier than %v", minstart)
	}

	ctx, err := buildContext(config)

	if err != nil {
		return errors.Annotate(err, "Failed to build context")
	}

	daysToProcess := int(endAt.Sub(startAt))

	if daysToProcess < 0 {
		return errors.New("startAt is after endAt")
	}

	logrus.Infof("Processing %v days", daysToProcess)

	for day := 0; day < daysToProcess; day++ {
		err = DownloadForDay(startAt.Add(date.PeriodOfDays(day)), ctx)
		if err != nil {
			return errors.Annotatef(err, "Failed to download for day %v", startAt.Add(date.PeriodOfDays(day)))
		}
	}

	return nil
}

func buildContext(config *config.Config) (*DownloadingContext, error) {

	client, err := flickraccess.NewDownloadClient(config)

	if err != nil {
		return nil, errors.Annotatef(err, "Failed to create flickr client")
	}

	return &DownloadingContext{
		flickrclient: client,
		config: config,
	}, nil
}

func DownloadForDay(day date.Date, ctx *DownloadingContext) error {

	logrus.Infof("Downloading for day %v", day.Format("2006-01-02"))

	//first query for all photos that day
	batch := ctx.flickrclient.Search(day, day.Add(1))

	errorChannel := make(chan error)

	photoCount := 0

	for {

		photo, err := batch.NextPhoto()

		if err != nil {
			return errors.Annotate(err, "Failed to load photo")
		}

		if photo == nil {
			break
		}

		photoCount += 1
		go func() {
			errorChannel <- processPhoto(photo, ctx)
		}()

		if err != nil {
			return errors.Annotate(err, "Failed to process photo")
		}
	}

	var rv error

	for i := 0; i < photoCount; i++ {
		select {
		case e := <-errorChannel:
			if e != nil {
				logrus.Errorf("Failed to process photo: %v", e)
				rv = errors.Errorf("Failures occurred when processing photos for day %v, check logs", day.Format("2006-01-02"))
			}
		}
	}

	logrus.Debugf("Completed downloading for day %v", day.Format("2006-01-02"))


	return rv
}

func processPhoto(photo *flickraccess.RemotePhoto, ctx *DownloadingContext) error {

	photoCtx := NewPhotoContext(ctx)

	photoCtx.SetRemote(photo)

	logrus.Debugf("Processing photo %v", photo.ID())

	err := getFilepath(photoCtx)

	if err != nil {
		return errors.Annotatef(err, "Failed to determine file path for photo: %v", err)
	}

	err = downloadAndWriteData(photoCtx)

	if err != nil {
		return errors.Annotatef(err, "Failed to write date for photo: %v", err)
	}

	return nil
}

func downloadAndWriteData(photoCtx *PhotoContext) error {
	//write meta
	meta, err := photoCtx.Photo.GetMeta()

	if err != nil {
		return errors.Annotatef(err, "Failed to get metadata: %v", photoCtx.Photo.ID())
	}

	asJson, err := json.Marshal(meta)

	if err != nil {
		return errors.Annotatef(err, "Failed to marshal metadata: %v", meta.Title)
	}

	logrus.Debugf("Writing metadata for %v to %v", meta.Title, photoCtx.Filepath + ".meta")
	err = ioutil.WriteFile(photoCtx.Filepath + ".meta", asJson, 0666)

	if err != nil {
		return errors.Annotatef(err, "Failed to write meta file: %v", meta.Title)
	}

	//
	urlToFetch := ""

	for i := range meta.SizeList {
		if meta.SizeList[i].Label == "Original" {
			urlToFetch = meta.SizeList[i].Source
		}
	}

	if urlToFetch == "" {
		return errors.Errorf("Failed to find original download URL for photo %s", meta.Title)
	}

	urlObj, err := url.Parse(urlToFetch)
	fileextension := "jpg"

	if err != nil {
		return errors.Annotate(err,"Failed to parse URL for photo")
	}

	segments := strings.Split(urlObj.Path, "/")
	parts := strings.Split(segments[len(segments) - 1], ".")

	if len(parts) < 2 {
		logrus.Warnf("Failed to detect file path from url, defaulting to 'jpg': %v", urlToFetch)
	} else {
		fileextension = parts[len(parts) - 1]
	}

	logrus.Debugf("Writing file for %v to %v", meta.Title, photoCtx.Filepath + "." + fileextension)
	err = DownloadFile(photoCtx.Filepath + "." + fileextension, urlToFetch)

	if err != nil {
		return errors.Annotatef(err, "Failed to download file: %v", err)
	}

	return nil

}

func getFilepath(photoCtx *PhotoContext) error {

	toDir := photoCtx.DownloadingContext.config.ArchiveDir
	meta, err := photoCtx.Photo.GetMeta()

	if err != nil {
		return errors.Annotate(err, "Failed to load metadata for photo")
	}

	date, err := time.Parse(flicrkDateFormat, meta.Dates.Taken)

	if err != nil {
		return errors.Annotatef(err, "Failed to parse date taken from flickr: %v", meta.Dates.Taken)
	}

	filename := meta.Title
	subdir := ""

	if meta.Sets != nil && len(meta.Sets) > 0 {
		subdir = meta.Sets[0].Title

		if len(meta.Sets) > 1 {
			logrus.Warn("Multiple sets detected for photo %v / %v", meta.Id, meta.Title)
		}
	}

	targetDir := fmt.Sprintf("%v/%v/%.2d/%v", toDir, date.Year(), date.Month(), subdir)
	err = os.MkdirAll(targetDir, 0755)

	if err != nil {
		return errors.Trace(err)
	}

	newName := fmt.Sprintf("%v/%v", targetDir, filename)

	photoCtx.Filepath = newName

	return nil
}

type DownloadingContext struct {
	flickrclient *flickraccess.FlickrDownloadClient
	config       *config.Config
}


type PhotoContext struct {
	DownloadingContext *DownloadingContext
	Photo *flickraccess.RemotePhoto
	Filepath string
}

func NewPhotoContext(DownloadingContext *DownloadingContext) *PhotoContext {
	return &PhotoContext{
		DownloadingContext: DownloadingContext,
	}
}

func (pc *PhotoContext) SetRemote(photo *flickraccess.RemotePhoto) {
	pc.Photo = photo
}

func DownloadFile(filepath string, url string) error {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
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
)

import "github.com/jpg0/flickrdown/config"

var minstart = time.Date(2000, 0, 0, 0, 0, 0, 0, time.UTC)
const flicrkDateFormat = "2006-01-02 15:04:05"

func BeginBatchDownload(startAt time.Time, endAt time.Time, config *config.Config) error {

	logrus.Debugf("Beginning batch download")


	if !isDate(startAt) {
		return errors.New("startDate is not a date (has a time component)")
	}

	if startAt.Before(minstart) {
		return errors.Errorf("Cannot begin batch earlier than %v", minstart)
	}

	if !isDate(endAt) {
		return errors.New("endDate is not a date (has a time component)")
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
		config: config,
	}, nil
}

func DownloadForDay(day time.Time, ctx *DownloadingContext) error {

	logrus.Debugf("Downloading for day %v", day.Format("2006-01-02"))

	//first query for all photos that day
	batch := ctx.flickrclient.Search(day, day.AddDate(0, 0, 1))

	for {

		photoCtx := NewPhotoContext(ctx)

		photo, err := batch.NextPhoto()

		if err != nil {
			return errors.Annotate(err, "Failed to load photo")
		}

		photoCtx.SetRemote(photo)

		logrus.Debugf("Processing photo %v", photo.ID())

		err = processPhoto(photoCtx)

		if err != nil {
			return errors.Annotate(err, "Failed to process photo")
		}

	//	if photo == nil {
			break
	//	}
	}

	return nil
}

func processPhoto(photoCtx *PhotoContext) error {

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

	err = ioutil.WriteFile(photoCtx.Filepath + ".meta", asJson, 0666)

	if err != nil {
		return errors.Annotatef(err, "Failed to write meta file: %v", meta.Title)
	}

	fmt.Println(string(asJson))

	//
	urlToFetch := ""

	for i := range meta.SizeList {
		if meta.SizeList[i].Label == "Original" {
			urlToFetch = meta.SizeList[i].Source
		}
	}

	if urlToFetch == "" {
		return errors.Errorf("Failed to find original download URL for photo %v", meta.Title)
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

func isDate(time time.Time) bool {
	h, m, s := time.Clock()
	return h == 0 && m == 0 && s == 0
}

type DownloadingContext struct {
	flickrclient *flickraccess.FlickrDownloadClient
	config *config.Config
}

type SavedState struct {
	startAt time.Time
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
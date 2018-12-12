package flickraccess

import (
	"encoding/json"
	"encoding/xml"
	"github.com/jpg0/flickr"
	"github.com/jpg0/flickr/photos"
	"github.com/jpg0/flickrdown/config"
	"github.com/juju/errors"
	"time"
)

type FlickrDownloadClient struct {
	client *flickr.FlickrClient
}

func NewDownloadClient(config *config.Config) (*FlickrDownloadClient, error) {
	client := flickr.NewFlickrClient(config.APIKey, config.SharedSecret)
	token, err := getToken(client)

	if err != nil {
		return nil, err
	}

	client.OAuthToken = token.OAuthToken
	client.OAuthTokenSecret = token.OAuthTokenSecret

	return &FlickrDownloadClient{client: client}, nil
}

func (downloadclient *FlickrDownloadClient) Search(min_upload_date time.Time, max_upload_date time.Time) *DownloadBatch {

	return &DownloadBatch{
		from:   min_upload_date,
		to:     max_upload_date,
		client: downloadclient.client,
	}
}

type DownloadBatch struct {
	from     time.Time
	to       time.Time
	response *photos.PhotoSearchResponse
	client   *flickr.FlickrClient
	cursor   int
}

func (batch *DownloadBatch) NextPhoto() (*RemotePhoto, error) {
	if err := batch.prepare(); err != nil {
		return nil, err
	}

	batch.cursor++

	return &RemotePhoto{
		photoInfo: batch.response.PhotoList.Photos[batch.cursor-1],
		client:    batch.client,
	}, nil
}

func (batch *DownloadBatch) prepare() error {

	if batch.response == nil {

		response, err := photos.Search(batch.client, true, "me", batch.from, batch.to)

		if err != nil {
			return errors.Annotate(err, "Failed to search for photos")
		}

		batch.response = response
	}

	return nil
}

type RemotePhoto struct {
	photoInfo photos.PhotoInfo
	client    *flickr.FlickrClient
}

func (remotePhoto *RemotePhoto) ID() string {
	return remotePhoto.photoInfo.Id
}

func (remotePhoto *RemotePhoto) GetMeta() (*Meta, error) {

	photoInfoResponse, err := photos.GetInfo(remotePhoto.client, remotePhoto.photoInfo.Id, "")

	if err != nil {
		return nil, errors.Annotate(err, "Failed to retrieve photo info")
	}

	photoAllContextsResponse, err := photos.GetAllContexts(remotePhoto.client, remotePhoto.photoInfo.Id, "")

	if err != nil {
		return nil, errors.Annotate(err, "Failed to retrieve photo context")
	}

	meta := &Meta{
		photoInfoResponse.Photo,
		photoAllContextsResponse.PhotoAllContexts,
	}

	return meta, nil
}

type Meta struct {
	photos.PhotoInfo
	photos.PhotoAllContexts
}

func getJson(client *flickr.FlickrClient, method string, id string) (*flickr.BasicResponse, error) {
	client.Init()
	client.EndpointUrl = flickr.API_ENDPOINT
	client.HTTPVerb = "POST"
	client.Args.Set("method", "flickr.photos.getInfo")
	client.Args.Set("photo_id", id)
	client.OAuthSign()

	response := &flickr.BasicResponse{}
	err := flickr.DoPost(client, response)
	return response, err
}

func mergeXmlsToJson(json1 string, json2 string) (string, error) {
	out1 := map[string]interface{}{}
	err := xml.Unmarshal([]byte(json1), &out1)

	if err != nil {
		return "", errors.Annotate(err, "Failed to load xml")
	}

	out2 := map[string]interface{}{}
	err = xml.Unmarshal([]byte(json2), &out2)

	if err != nil {
		return "", errors.Annotate(err, "Failed to load xml")
	}

	for k, v := range out2 {
		out1[k] = v
	}

	rv, err := json.Marshal(out1)

	if err != nil {
		return "", errors.Annotate(err, "Failed to serialise to json")
	}

	return string(rv), nil
}
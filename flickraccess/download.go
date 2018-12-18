package flickraccess

import (
	"encoding/json"
	"encoding/xml"
	"github.com/Sirupsen/logrus"
	"github.com/jpg0/flickr"
	"github.com/jpg0/flickr/photos"
	"github.com/jpg0/flickrdown/config"
	"github.com/juju/errors"
	"github.com/rickb777/date"
)

type FlickrDownloadClient struct {
	apikey string
	sharedsecret string
}

func NewDownloadClient(config *config.Config) (*FlickrDownloadClient, error) {
	return &FlickrDownloadClient{
		apikey: config.APIKey,
		sharedsecret: config.SharedSecret,
	}, nil
}

func (downloadclient *FlickrDownloadClient) newClient() *flickr.FlickrClient {
	client := flickr.NewFlickrClient(downloadclient.apikey, downloadclient.sharedsecret)
	token, err := getToken(client)

	if err != nil { //lazy
		panic("Failed to build flickr client")
	}

	client.OAuthToken = token.OAuthToken
	client.OAuthTokenSecret = token.OAuthTokenSecret

	return client
}

func (downloadclient *FlickrDownloadClient) Search(min_upload_date date.Date, max_upload_date date.Date) *DownloadBatch {

	return &DownloadBatch{
		from:   min_upload_date,
		to:     max_upload_date,
		client: downloadclient,
	}
}

type DownloadBatch struct {
	from     date.Date
	to       date.Date
	response *photos.PhotoSearchResponse
	client   *FlickrDownloadClient
	cursor   int
}

func (batch *DownloadBatch) NextPhoto() (*RemotePhoto, error) {


	//if not yet fetched
	if batch.response == nil {

		logrus.Debugf("Searching for photos from %v to %v", batch.from, batch.to)

		response, err := photos.Search(batch.client.newClient(), true, "me", batch.from.UTC(), batch.to.UTC(), 1)

		if err != nil {
			return nil, errors.Annotate(err, "Failed to search for photos")
		}

		logrus.Debugf("%v results for photos from %v to %v", len(response.PhotoList.Photos), batch.from, batch.to)

		batch.response = response
	}

	//if exhausted
	if batch.cursor == len(batch.response.PhotoList.Photos) {
		//get next batch or error
		if batch.response.PhotoList.Page < batch.response.PhotoList.Pages {
			response, err := photos.Search(batch.client.newClient(), true, "me", batch.from.UTC(), batch.to.UTC(), batch.response.PhotoList.Page + 1)

			if err != nil {
				return nil, errors.Annotate(err, "Failed to get next search page for photos")
			}

			batch.response = response
			batch.cursor = 0

		} else { // no more results
			return nil, nil
		}
	}

	batch.cursor++

	return &RemotePhoto{
		photoInfo: batch.response.PhotoList.Photos[batch.cursor-1],
		client:    batch.client.newClient(),
	}, nil
}

type RemotePhoto struct {
	photoInfo photos.PhotoInfo
	client    *flickr.FlickrClient
	meta *Meta
}

func (remotePhoto *RemotePhoto) ID() string {
	return remotePhoto.photoInfo.Id
}

func (remotePhoto *RemotePhoto) GetMeta() (*Meta, error) {

	if remotePhoto.meta == nil {
		photoInfoResponse, err := photos.GetInfo(remotePhoto.client, remotePhoto.photoInfo.Id, "")

		if err != nil {
			return nil, errors.Annotate(err, "Failed to retrieve photo info")
		}

		photoAllContextsResponse, err := photos.GetAllContexts(remotePhoto.client, remotePhoto.photoInfo.Id, "")

		if err != nil {
			return nil, errors.Annotate(err, "Failed to retrieve photo context")
		}

		photoSizesResponse, err := photos.GetSizes(remotePhoto.client, remotePhoto.photoInfo.Id, "")

		if err != nil {
			return nil, errors.Annotate(err, "Failed to retrieve photo sizes")
		}

		remotePhoto.meta = &Meta{
			photoInfoResponse.Photo,
			photoAllContextsResponse.PhotoAllContexts,
			photoSizesResponse.Sizes,

		}
	}

	return remotePhoto.meta, nil
}

type Meta struct {
	photos.PhotoInfo
	photos.PhotoAllContexts
	photos.PhotoSizes
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
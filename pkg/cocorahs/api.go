package cocorahs

import (
	"fmt"

	"github.com/jhot21/hats/internal/util"
	"github.com/go-resty/resty/v2"
)

type CocorahsClient struct {
	restClient *resty.Client
}

func New() *CocorahsClient {
	return &CocorahsClient{
		restClient: resty.New().
			SetBaseURL("https://dex.cocorahs.org").
			SetHeader("Accept", "application/json").
			SetHeader("User-Agent", "(HATS-client, hats@jhot.me)"),
	}
}

func (c *CocorahsClient) GetStationArchive(stationNumber string) (StationArchive, error) {
	var data StationArchive

	_, err := util.CheckSuccess(c.restClient.R().SetResult(&data).
		Get(fmt.Sprintf("resources/stations/%s/stationarchive", stationNumber)))

	return data, err
}

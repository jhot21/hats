package airresourcenet

import (
	"encoding/json"
	"fmt"
	"strings"

	"code.jhot.me/jhot/hats/internal/util"
	"github.com/go-resty/resty/v2"
)

type AirResourceNetClient struct {
	restClient *resty.Client
	SiteCode   string
}

func New(site string) *AirResourceNetClient {
	return &AirResourceNetClient{
		restClient: resty.New().
			SetBaseURL(fmt.Sprintf("https://www.air-resource.net/%s", strings.ToUpper(site))),
		SiteCode: site,
	}
}

func (c *AirResourceNetClient) GetAirQualityData() (AirQualityData, error) {
	var data AirQualityData
	// Can't use .SetResult because the content type is text/plain
	resp, err := util.CheckSuccess(c.restClient.R().Get(fmt.Sprintf("%sdatajson.txt", c.SiteCode)))
	if err != nil {
		return data, err
	}
	err = json.Unmarshal(resp.Body(), &data)
	return data, err
}

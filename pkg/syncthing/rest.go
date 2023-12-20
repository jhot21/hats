package syncthing

import (
	"fmt"

	"code.jhot.me/jhot/hats/internal/util"
	"github.com/go-resty/resty/v2"
)

type SyncthingClient struct {
	restClient *resty.Client
}

func New(host string, token string) *SyncthingClient {
	return &SyncthingClient{
		restClient: resty.New().
			SetBaseURL(fmt.Sprintf("%s/rest", host)).
			SetHeader("X-API-Key", token),
	}
}

func (c *SyncthingClient) GetVersion() (VersionInfo, error) {
	var data VersionInfo
	_, err := util.CheckSuccess(c.restClient.R().SetResult(&data).Get("system/version"))
	return data, err
}

func (c *SyncthingClient) GetDeviceStats() (map[string]DeviceStatistics, error) {
	var data map[string]DeviceStatistics
	_, err := util.CheckSuccess(c.restClient.R().SetResult(&data).Get("stats/device"))
	return data, err
}

func (c *SyncthingClient) Pause(id string) error {
	req := c.restClient.R()

	if id != "" {
		req.SetHeader("device", id)
	}

	_, err := util.CheckSuccess(req.Post("system/pause"))
	return err
}

func (c *SyncthingClient) Resume(id string) error {
	req := c.restClient.R()

	if id != "" {
		req.SetHeader("device", id)
	}

	_, err := util.CheckSuccess(req.Post("system/resume"))
	return err
}

func (c *SyncthingClient) GetFolderConfig(id string) ([]FolderConfig, error) {
	url := "config/folders"

	if id != "" {

		url = fmt.Sprintf("%s/%s", url, id)
		var data FolderConfig
		_, err := util.CheckSuccess(c.restClient.R().SetResult(&data).Get(url))
		return []FolderConfig{data}, err

	} else {

		var data []FolderConfig
		_, err := util.CheckSuccess(c.restClient.R().SetResult(&data).Get(url))
		return data, err

	}
}

func (c *SyncthingClient) SetFolderConfig(id string, param string, value any) error {
	_, err := util.CheckSuccess(c.restClient.R().SetBody(map[string]any{param: value}).Patch(fmt.Sprintf("config/folders/%s", id)))
	return err
}

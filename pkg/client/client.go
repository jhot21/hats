package client

import (
	"fmt"

	"code.jhot.me/jhot/hats/pkg/homeassistant"
	"github.com/go-resty/resty/v2"
)

type HatsClient struct {
	client *resty.Client
}

func NewHatsClient(baseUrl string) *HatsClient {
	client := resty.New().SetBaseURL(baseUrl)
	return &HatsClient{
		client: client,
	}
}

func (c *HatsClient) GetState(entityId string) (string, error) {
	resp, err := c.client.R().Get(fmt.Sprintf("api/state/%s", entityId))
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}

	if err != nil {
		return "", err
	}

	return resp.String(), nil
}

func (c *HatsClient) GetStateBool(entityId string) (bool, error) {
	stateString, err := c.GetState(entityId)

	if err != nil {
		return false, err
	}

	return homeassistant.StateToBool(stateString), nil
}

func (c *HatsClient) CallService(entityId string, service string, extras ...map[string]string) error {
	req := c.client.R()
	if len(extras) > 0 {
		data := map[string]interface{}{}
		for _, extra := range extras {
			for k, v := range extra {
				data[k] = v
			}
		}
		req.SetBody(data)
	}

	resp, err := req.Post(fmt.Sprintf("api/state/%s/%s", entityId, service))
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}

	if err != nil {
		return err
	}

	return nil
}

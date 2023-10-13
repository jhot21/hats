package client

import (
	"fmt"

	"code.jhot.me/jhot/hats/internal/api"
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

func (c *HatsClient) GetTimer(name string) (string, error) {
	resp, err := c.client.R().Get(fmt.Sprintf("api/timer/%s", name))
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}

	if err != nil {
		return "", err
	}

	return resp.String(), nil
}

func (c *HatsClient) SetTimer(name string, duration string, force bool) (string, error) {
	data := api.CreateTimerData{
		Duration: duration,
		Force:    force,
	}

	resp, err := c.client.R().SetBody(data).Post(fmt.Sprintf("api/timer/%s", name))
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}

	if err != nil {
		return "", err
	}

	return resp.String(), nil
}

func (c *HatsClient) DeleteTimer(name string) error {
	resp, err := c.client.R().Delete(fmt.Sprintf("api/timer/%s", name))
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}
	return err
}

func (c *HatsClient) GetSchedule(name string) (string, error) {
	resp, err := c.client.R().Get(fmt.Sprintf("api/schedule/%s", name))
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}

	if err != nil {
		return "", err
	}

	return resp.String(), nil
}

// SetSchedule: set a cron schedule
//
//	name: a unique identifying string
//	cron: a cron expression with seconds, like "0 */5 * * * *" (every 5 minutes)
func (c *HatsClient) SetSchedule(name string, cron string) (string, error) {
	data := api.CreateScheduleData{
		Cron: cron,
	}

	resp, err := c.client.R().SetBody(data).Post(fmt.Sprintf("api/schedule/%s", name))
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}

	if err != nil {
		return "", err
	}

	return resp.String(), nil
}

func (c *HatsClient) DeleteSchedule(name string) error {
	resp, err := c.client.R().Delete(fmt.Sprintf("api/schedule/%s", name))
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}
	return err
}

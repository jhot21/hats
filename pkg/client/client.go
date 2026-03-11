package client

import (
	"fmt"
	"strconv"

	"codeberg.org/jhot/hats/internal/api"
	ha "codeberg.org/jhot/hats/pkg/homeassistant"
	"codeberg.org/jhot/hats/pkg/ntfy"
	"github.com/go-resty/resty/v2"
)

type HatsClient struct {
	client *resty.Client
}

func NewHatsClient(baseUrl string, token string) *HatsClient {
	client := resty.New().SetBaseURL(baseUrl)
	if token != "" {
		client.SetHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	return &HatsClient{
		client: client,
	}
}

func (c *HatsClient) GetStateFull(entityId string) (ha.StateData, error) {
	var data ha.StateData
	resp, err := c.client.R().SetResult(&data).SetQueryParam("full", "true").Get(fmt.Sprintf("api/state/%s", entityId))
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}

	if err != nil {
		return data, err
	}

	return data, nil
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

func (c *HatsClient) GetStateWithDefault(entityId string, defaultValue string) string {
	state, err := c.GetState(entityId)
	if err != nil {
		return defaultValue
	}
	return state
}

func (c *HatsClient) GetStateBool(entityId string) (bool, error) {
	stateString, err := c.GetState(entityId)

	if err != nil {
		return false, err
	}

	return ha.StateToBool(stateString), nil
}

func (c *HatsClient) GetStateFloat(entityId string) (float64, error) {
	stateString, err := c.GetState(entityId)

	if err != nil {
		return 0.0, err
	}

	parsed, err := strconv.ParseFloat(stateString, 64)
	return parsed, err
}

func (c *HatsClient) CallService(entityId string, service string, extras ...map[string]any) error {
	req := c.client.R()
	if len(extras) > 0 {
		data := map[string]interface{}{}
		for _, extra := range extras {
			for k, v := range extra {
				switch k {
				case "domain":
					req.SetQueryParam(k, fmt.Sprintf("%v", v))
				default:
					data[k] = v
				}
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

// Start a timer
//
//	name: the name of the timer (should be unique)
//	duration: time.Duration string for how long the timer should last
//	force: if true, will start the timer over even if it is already running
func (c *HatsClient) StartTimer(name string, duration string, force bool) (string, error) {
	data := api.StartTimerData{
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

func (c *HatsClient) CancelTimer(name string) error {
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

func (c *HatsClient) SendNtfyMessage(data *ntfy.Message) error {
	var result struct {
		ID string `json:"id"`
	}
	resp, err := c.client.R().SetBody(data).SetResult(&result).Post("api/ntfy")
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}
	if err != nil {
		return err
	}
	data.ID = result.ID
	return nil
}

func (c *HatsClient) DeleteNtfyMessage(data *ntfy.Message) error {
	resp, err := c.client.R().SetBody(data).Delete("api/ntfy")
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}
	return err
}

func (c *HatsClient) ClearNtfyMessage(data *ntfy.Message) error {
	resp, err := c.client.R().SetBody(data).Put("api/ntfy/clear")
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}
	return err
}

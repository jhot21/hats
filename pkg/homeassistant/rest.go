package homeassistant

import (
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
)

type RestClient struct {
	client *resty.Client
}

func NewRestClient(baseUrl, token string) *RestClient {
	client := resty.New().SetBaseURL(baseUrl)
	client.SetHeaders(map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
		"Accept":        "application/json",
	})
	return &RestClient{
		client: client,
	}
}

func (c *RestClient) GetState(entityId string) (StateData, error) {
	var data StateData
	resp, err := c.client.R().SetResult(&data).Get(fmt.Sprintf("api/states/%s", entityId))
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}
	return data, err
}

func (c *RestClient) CallService(entityId string, service string, extras ...map[string]any) error {
	domain := strings.Split(entityId, ".")[0]
	data := map[string]any{
		"entity_id": entityId,
	}
	for _, extra := range extras {
		for k, v := range extra {
			data[k] = v
		}
	}
	resp, err := c.client.R().SetBody(data).Post(fmt.Sprintf("api/services/%s/%s", domain, service))
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}
	return err
}

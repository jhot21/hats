package homeassistant

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	rsty "codeberg.org/jhot/hats/pkg/resty"
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
	return c.CallServiceManual(domain, entityId, service, extras...)
}

func (c *RestClient) CallServiceManual(domain string, entityId string, service string, extras ...map[string]any) error {
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

type CallServiceInput struct {
	EntityID string
	Service  string
	Extras   map[string]any
}

func (c *RestClient) CallServices(inputs ...*CallServiceInput) error {
	errorChannel := make(chan error)
	var wg sync.WaitGroup

	wg.Add(len(inputs))

	go func() {
		wg.Wait()
		close(errorChannel)
	}()

	for _, input := range inputs {
		go func(input *CallServiceInput) {
			defer wg.Done()
			err := c.CallService(input.EntityID, input.Service, input.Extras)
			if err != nil {
				errorChannel <- fmt.Errorf("error calling service %s for %s: %w", input.Service, input.EntityID, err)
			}
		}(input)
	}

	returnErrors := []error{}
	for err := range errorChannel {
		returnErrors = append(returnErrors, err)
	}
	return errors.Join(returnErrors...)
}

// Start time of history results. If not set, API defaults to 1 day before now
func HistoryStartTimestamp(value time.Time) rsty.RestyOption {
	return func(r *resty.Request) {
		// YYYY-MM-DDThh:mm:ssTZD
		r.SetPathParam("timestamp", value.Format(time.RFC3339))
	}
}

// End time of history results. If not set, API defaults to 1 day after start
func HistoryEndTimestamp(value time.Time) rsty.RestyOption {
	return func(r *resty.Request) {
		// YYYY-MM-DDThh:mm:ssTZD
		r.SetQueryParam("end_time", value.Format(time.RFC3339))
	}
}

func HistoryEntityFilter(entityIds ...string) rsty.RestyOption {
	return func(r *resty.Request) {
		r.SetQueryParam("filter_entity_id", strings.Join(entityIds, ","))
	}
}

func HistoryMinimalResponse() rsty.RestyOption {
	return func(r *resty.Request) {
		r.SetQueryParam("minimal_response", "true")
	}
}

func HistoryNoAttributes() rsty.RestyOption {
	return func(r *resty.Request) {
		r.SetQueryParam("no_attributes", "true")
	}
}

func HistorySignificantOnly() rsty.RestyOption {
	return func(r *resty.Request) {
		r.SetQueryParam("significant_changes_only", "true")
	}
}

func (c *RestClient) GetHistory(opts ...rsty.RestyOption) (HistoryResponse, error) {
	req := c.client.R()
	urlPath := "/api/history/period"

	for _, opt := range opts {
		opt(req)
	}

	if _, isSet := req.PathParams["timestamp"]; isSet {
		urlPath += "/{timestamp}"
	}

	var data HistoryResponse

	resp, err := req.SetResult(&data).Get(urlPath)
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}
	return data, err
}

package amcrest

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"code.jhot.me/jhot/hats/internal/util"
	"github.com/go-resty/resty/v2"
)

type AmcrestClient struct {
	restClient *resty.Client
}

func New(host string, username string, password string) *AmcrestClient {
	return &AmcrestClient{
		restClient: resty.New().SetBaseURL(host).SetDigestAuth(username, password),
	}
}

type GetSnapshotResponse struct {
	MimeType string
	Data     []byte
}

// Get a snapshot from the specified channel
//
// Parameter
//
//	channel int (optional): the channel index (starts at 1)
func (c *AmcrestClient) GetSnapshot(channel ...int) (GetSnapshotResponse, error) {
	cameraChannel := 1
	if len(channel) > 0 && channel[0] != 0 {
		cameraChannel = channel[0]
	}

	resp, err := util.CheckSuccess(c.restClient.R().SetQueryParam("channel", fmt.Sprintf("%d", cameraChannel)).Get("cgi-bin/snapshot.cgi"))
	if err != nil {
		return GetSnapshotResponse{}, err
	}

	return GetSnapshotResponse{
		MimeType: resp.Header().Get("content-type"),
		Data:     resp.Body(),
	}, nil
}

type ConfigValue struct {
	// Category[Channel].FormatType[FormatEncodeType].Param=Value
	Category         string
	Channel          int // The channel index (starts at 0)
	FormatType       string
	FormatEncodeType int
	Param            string
	Value            any
}

func (c ConfigValue) ToParamString() string {
	return fmt.Sprintf("%s[%d].%s[%d].%s", c.Category, c.Channel, c.FormatType, c.FormatEncodeType, c.Param)
}

// Parse config values from a string
//
//	input: a string like "Category[Channel].FormatType[FormatEncodeType].Param.Name=Value"
func ConfigValueFromString(input string) (ConfigValue, error) {
	output := ConfigValue{}
	values := strings.SplitN(input, "=", 2)
	if len(values) > 1 {
		output.Value = values[1]
	}

	sections := strings.SplitN(values[0], ".", 3)

	for i, section := range sections {
		if i < 2 {
			parts := strings.Split(section, "[")
			var val int64 = 0
			if len(parts) > 1 {
				var err error
				val, err = strconv.ParseInt(strings.TrimSuffix(parts[1], "]"), 10, 64)
				if err != nil {
					return output, fmt.Errorf("error parsing integer from string: %w", err)
				}
			}
			switch i {
			case 0:
				output.Category = parts[0]
				output.Channel = int(val)
			case 1:
				output.FormatType = parts[0]
				output.FormatEncodeType = int(val)
			}
		} else {
			output.Param = section
		}
	}

	return output, nil
}

func (c *AmcrestClient) SetConfig(inputs ...ConfigValue) error {
	if len(inputs) == 0 {
		return errors.New("no inputs provided")
	}

	req := c.restClient.R().SetQueryParam("action", "setConfig")

	for _, input := range inputs {
		req.SetQueryParam(input.ToParamString(), fmt.Sprintf("%v", input.Value))
	}

	_, err := util.CheckSuccess(req.Get("cgi-bin/configManager.cgi"))
	return err
}

func (c *AmcrestClient) GetEncodingConfigCapabilities() ([]ConfigValue, error) {
	req := c.restClient.R().SetQueryParam("action", "getConfigCaps")

	resp, err := util.CheckSuccess(req.Get("cgi-bin/encode.cgi"))
	values := []ConfigValue{}

	if err != nil {
		return values, err
	}
	for _, line := range strings.Split(string(resp.Body()), "\n") {
		val, err := ConfigValueFromString(line)
		if err != nil {
			return values, fmt.Errorf("error parsing line: %w", err)
		}
		values = append(values, val)
	}

	return values, nil
}

func (c *AmcrestClient) GetConfig(configName string) ([]ConfigValue, error) {
	resp, err := util.CheckSuccess(c.restClient.R().
		SetQueryParam("action", "getConfig").SetQueryParam("name", configName).
		Get("cgi-bin/configManager.cgi"))
	values := []ConfigValue{}

	if err != nil {
		return values, err
	}
	for _, line := range strings.Split(strings.ReplaceAll(string(resp.Body()), "\r\n", "\n"), "\n") {
		if line == "" {
			continue
		}
		val, err := ConfigValueFromString(strings.TrimPrefix(line, "table."))
		if err != nil {
			return values, fmt.Errorf("error parsing line: %w", err)
		}
		values = append(values, val)
	}

	return values, nil
}

package ntfy

import (
	"fmt"

	"codeberg.org/jhot/hats/pkg/config"
	ntfyPkg "codeberg.org/jhot/hats/pkg/ntfy"
	"github.com/go-resty/resty/v2"
)

var (
	ntfyClient *resty.Client
)

func InitClient(cfg *config.HatsConfig) {
	ntfyClient = resty.New().SetBaseURL(cfg.NtfyHost)
	if cfg.NtfyToken != "" {
		ntfyClient.SetHeader("Authorization", fmt.Sprintf("Bearer %s", cfg.NtfyToken))
	}
}

func Send(data *ntfyPkg.Message) error {
	if data.ID == "" {
		var result struct {
			ID string `json:"id"`
		}
		resp, err := ntfyClient.R().SetBody(data).SetResult(&result).Post("")
		if err != nil {
			return err
		}
		if !resp.IsSuccess() {
			return fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
		}
		data.ID = result.ID
		return nil
	}
	resp, err := ntfyClient.R().SetBody(data).Put(fmt.Sprintf("%s/%s", data.Topic, data.ID))
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}
	return err
}

func Delete(data *ntfyPkg.Message) error {
	if data.ID == "" || data.Topic == "" {
		return fmt.Errorf("ntfy: id and topic are required to delete a message")
	}
	resp, err := ntfyClient.R().Delete(fmt.Sprintf("%s/%s", data.Topic, data.ID))
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}
	return err
}

func Clear(data *ntfyPkg.Message) error {
	if data.ID == "" || data.Topic == "" {
		return fmt.Errorf("ntfy: id and topic are required to clear a message")
	}
	resp, err := ntfyClient.R().Put(fmt.Sprintf("%s/%s/clear", data.Topic, data.ID))
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}
	return err
}

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

func Send(data ntfyPkg.Message) error {
	resp, err := ntfyClient.R().SetBody(data).Post("")
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}
	return err
}

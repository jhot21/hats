package ntfy

import (
	"fmt"

	"codeberg.org/jhot/hats/internal/secrets"
	ntfyPkg "codeberg.org/jhot/hats/pkg/ntfy"
	"github.com/go-resty/resty/v2"
)

// Client sends ntfy notifications, reading credentials from the secret store at call time.
// This means credential changes via the admin UI take effect immediately without a restart.
type Client struct {
	store     secrets.SecretStore
	baseHost  string
	baseToken string
}

// New creates a Client. baseHost and baseToken are the env-var defaults; the store overrides them.
func New(store secrets.SecretStore, baseHost, baseToken string) *Client {
	return &Client{store: store, baseHost: baseHost, baseToken: baseToken}
}

// resolveCredentials returns the effective host and token, preferring store values.
func (c *Client) resolveCredentials() (host, token string) {
	host = c.baseHost
	token = c.baseToken
	if s, err := c.store.Get("NTFY_HOST"); err == nil {
		host = s.Value
	}
	if s, err := c.store.Get("NTFY_TOKEN"); err == nil {
		token = s.Value
	}
	return
}

func (c *Client) buildHTTPClient() *resty.Client {
	host, token := c.resolveCredentials()
	rc := resty.New().SetBaseURL(host)
	if token != "" {
		rc.SetHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	return rc
}

func (c *Client) Send(data *ntfyPkg.Message) error {
	rc := c.buildHTTPClient()
	if data.ID == "" {
		var result struct {
			ID string `json:"id"`
		}
		resp, err := rc.R().SetBody(data).SetResult(&result).Post("")
		if err != nil {
			return err
		}
		if !resp.IsSuccess() {
			return fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
		}
		data.ID = result.ID
		return nil
	}
	resp, err := rc.R().SetBody(data).Put("")
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}
	return err
}

func (c *Client) Delete(data *ntfyPkg.Message) error {
	if data.ID == "" || data.Topic == "" {
		return fmt.Errorf("ntfy: id and topic are required to delete a message")
	}
	rc := c.buildHTTPClient()
	resp, err := rc.R().Delete(fmt.Sprintf("%s/%s", data.Topic, data.ID))
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}
	return err
}

func (c *Client) Clear(data *ntfyPkg.Message) error {
	if data.ID == "" || data.Topic == "" {
		return fmt.Errorf("ntfy: id and topic are required to clear a message")
	}
	rc := c.buildHTTPClient()
	resp, err := rc.R().Put(fmt.Sprintf("%s/%s/clear", data.Topic, data.ID))
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status code received: %s", resp.StatusCode(), resp.String())
	}
	return err
}

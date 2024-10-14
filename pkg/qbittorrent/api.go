package qbittorrent

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"codeberg.org/jhot/hats/internal/util"
	"github.com/go-resty/resty/v2"
)

type QbittorrentClient struct {
	restClient *resty.Client
}

func New(host string) *QbittorrentClient {
	return &QbittorrentClient{
		restClient: resty.New().SetTimeout(30 * time.Second).SetBaseURL(fmt.Sprintf("%s/api/v2", host)),
	}
}

func (c *QbittorrentClient) Login(user string, pass string) error {
	resp, err := util.CheckSuccess(c.restClient.R().SetFormData(map[string]string{
		"username": user,
		"password": pass,
	}).Post("auth/login"))

	if err != nil {
		return err
	}

	authCookie := resp.Header().Get("set-cookie")
	if authCookie == "" {
		return fmt.Errorf("auth cookie not found.\nResponse: %s\nHeaders: %+v", resp.Body(), resp.Header())
	}

	first := strings.Split(authCookie, ";")[0]
	parts := strings.Split(first, "=")
	if len(parts) != 2 {
		return fmt.Errorf("auth cookie not in expected format: %s", authCookie)
	}

	c.restClient.SetCookie(&http.Cookie{
		Name:  parts[0],
		Value: parts[1],
	})

	return nil
}

func (c *QbittorrentClient) Logout() error {
	_, err := util.CheckSuccess(c.restClient.R().Get("auth/logout"))

	c.restClient.SetCookies([]*http.Cookie{})

	return err
}

func (c *QbittorrentClient) GetVersion() (string, error) {
	resp, err := util.CheckSuccess(c.restClient.R().Get("app/version"))
	if err != nil {
		return "", err
	}

	return string(resp.Body()), nil
}

func (c *QbittorrentClient) GetTransferInfo() (GlobalTransferInfo, error) {
	var data GlobalTransferInfo
	_, err := util.CheckSuccess(c.restClient.R().SetResult(&data).Get("transfer/info"))
	return data, err
}

func (c *QbittorrentClient) GetAltSpeedLimitState() (bool, error) {
	resp, err := util.CheckSuccess(c.restClient.R().Get("transfer/speedLimitsMode"))
	if err != nil {
		return false, err
	}

	return string(resp.Body()) == "1", nil
}

func (c *QbittorrentClient) ToggleAltSpeedLimitState() error {
	_, err := util.CheckSuccess(c.restClient.R().Post("transfer/toggleSpeedLimitsMode"))
	return err
}

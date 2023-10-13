package config

import (
	"fmt"
	"strconv"

	"code.jhot.me/jhot/hats/internal/util"
)

type HatsConfig struct {
	HomeAssistantHost         string
	HomeAssistantPort         string
	HomeAssistantSecure       bool
	HomeAssistantBaseUrl      string
	HomeAssistantWebsocketUrl string
	HomeAssistantToken        string

	NatsHost       string
	NatsPort       string
	NatsBaseUrl    string
	NatsToken      string
	NatsClientName string

	HatsHost    string
	HatsPort    string
	HatsSecure  bool
	HatsBaseUrl string
}

func FromEnvironment() *HatsConfig {
	config := &HatsConfig{
		HomeAssistantHost:  util.GetEnvWithDefault("HASS_HOST", "127.0.0.1"),
		HomeAssistantPort:  util.GetEnvWithDefault("HASS_PORT", "8123"),
		HomeAssistantToken: util.GetEnvWithDefault("HASS_TOKEN", ""),
		NatsHost:           util.GetEnvWithDefault("NATS_HOST", "127.0.0.1"),
		NatsPort:           util.GetEnvWithDefault("NATS_PORT", "4222"),
		NatsToken:          util.GetEnvWithDefault("NATS_TOKEN", ""),
		NatsClientName:     util.GetEnvWithDefault("NATS_CLIENT_NAME", "hats"),
		HatsHost:           util.GetEnvWithDefault("HATS_HOST", "hats"),
		HatsPort:           util.GetEnvWithDefault("HATS_PORT", "8888"),
	}

	config.HomeAssistantSecure, _ = strconv.ParseBool(util.GetEnvWithDefault("HASS_SECURE", "false"))
	hassProtocol := "http"
	HassWsProtocol := "ws"
	if config.HomeAssistantSecure {
		hassProtocol += "s"
		HassWsProtocol += "s"
	}
	config.HomeAssistantBaseUrl = fmt.Sprintf("%s://%s:%s", hassProtocol, config.HomeAssistantHost, config.HomeAssistantPort)
	config.HomeAssistantWebsocketUrl = fmt.Sprintf("%s://%s:%s/api/websocket", HassWsProtocol, config.HomeAssistantHost, config.HomeAssistantPort)

	config.NatsBaseUrl = fmt.Sprintf("nats://%s:%s", config.NatsHost, config.NatsPort)

	config.HatsSecure, _ = strconv.ParseBool(util.GetEnvWithDefault("HATS_SECURE", "false"))
	hatsProtocol := "http"
	if config.HatsSecure {
		hatsProtocol += "s"
	}
	config.HatsBaseUrl = fmt.Sprintf("%s://%s:%s", hatsProtocol, config.HatsHost, config.HatsPort)

	return config
}

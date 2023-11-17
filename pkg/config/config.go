package config

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"code.jhot.me/jhot/hats/internal/util"
)

type HatsConfig struct {
	LogLevl string

	HomeAssistantHost   string
	HomeAssistantPort   string
	HomeAssistantSecure bool
	HomeAssistantToken  string

	NatsHost       string
	NatsPort       string
	NatsToken      string
	NatsClientName string

	HatsHost   string
	HatsPort   string
	HatsToken  string
	HatsSecure bool

	NtfyHost  string
	NtfyToken string
}

func FromEnvironment() *HatsConfig {
	config := &HatsConfig{
		LogLevl: util.GetEnvWithDefault("LOG_LEVEL", "INFO"),

		HomeAssistantHost:  util.GetEnvWithDefault("HASS_HOST", "127.0.0.1"),
		HomeAssistantPort:  util.GetEnvWithDefault("HASS_PORT", "8123"),
		HomeAssistantToken: util.GetEnvWithDefault("HASS_TOKEN", ""),

		NatsHost:       util.GetEnvWithDefault("NATS_HOST", "127.0.0.1"),
		NatsPort:       util.GetEnvWithDefault("NATS_PORT", "4222"),
		NatsToken:      util.GetEnvWithDefault("NATS_TOKEN", ""),
		NatsClientName: util.GetEnvWithDefault("NATS_CLIENT_NAME", "hats"),

		HatsHost:  util.GetEnvWithDefault("HATS_HOST", "hats"),
		HatsPort:  util.GetEnvWithDefault("HATS_PORT", "8888"),
		HatsToken: util.GetEnvWithDefault("HATS_TOKEN", ""),

		NtfyHost:  util.GetEnvWithDefault("NTFY_HOST", "https://ntfy.sh"),
		NtfyToken: util.GetEnvWithDefault("NTFY_TOKEN", ""),
	}

	config.HomeAssistantSecure, _ = strconv.ParseBool(util.GetEnvWithDefault("HASS_SECURE", "false"))
	config.HatsSecure, _ = strconv.ParseBool(util.GetEnvWithDefault("HATS_SECURE", "false"))

	return config
}

func (c *HatsConfig) GetHomeAssistantBaseUrl() string {
	hassProtocol := "http"
	if c.HomeAssistantSecure {
		hassProtocol += "s"
	}
	return fmt.Sprintf("%s://%s:%s", hassProtocol, c.HomeAssistantHost, c.HomeAssistantPort)
}

func (c *HatsConfig) GetHomeAssistantWebsocketUrl() string {
	protocol := "ws"
	if c.HomeAssistantSecure {
		protocol += "s"
	}
	return fmt.Sprintf("%s://%s:%s/api/websocket", protocol, c.HomeAssistantHost, c.HomeAssistantPort)
}

func (c *HatsConfig) GetNatsBaseUrl() string {
	return fmt.Sprintf("nats://%s:%s", c.NatsHost, c.NatsPort)
}

func (c *HatsConfig) GetHatsBaseUrl() string {
	protocol := "http"
	if c.HomeAssistantSecure {
		protocol += "s"
	}
	return fmt.Sprintf("%s://%s:%s", protocol, c.HatsHost, c.HatsPort)
}

func (c *HatsConfig) GetLogLevel() slog.Level {
	switch strings.ToLower(c.LogLevl) {
	case "error":
		return slog.LevelError
	case "warn":
		return slog.LevelWarn
	case "debug":
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}

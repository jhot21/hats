package config

import (
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strconv"
	"strings"

	"codeberg.org/jhot/hats/pkg/client"
)

// HatsConfig holds configuration shared by the HATS server and all subscriber services.
type HatsConfig struct {
	LogLvl string `config:"LOG_LEVEL" default:"INFO"`

	HomeAssistantHost   string `config:"HASS_HOST" default:"127.0.0.1"`
	HomeAssistantPort   string `config:"HASS_PORT" default:"8123"`
	HomeAssistantSecure bool   `config:"HASS_SECURE" default:"false"`
	HomeAssistantToken  string `config:"HASS_TOKEN" default:""`

	NatsHost       string `config:"NATS_HOST" default:"127.0.0.1"`
	NatsPort       string `config:"NATS_PORT" default:"4222"`
	NatsToken      string `config:"NATS_TOKEN" default:""`
	NatsClientName string `config:"NATS_CLIENT_NAME" default:"hats"`

	HatsHost   string `config:"HATS_HOST" default:"hats"`
	HatsPort   string `config:"HATS_PORT" default:"8888"`
	HatsToken  string `config:"HATS_TOKEN" default:""`
	HatsSecure bool   `config:"HATS_SECURE" default:"false"`

	NtfyHost  string `config:"NTFY_HOST" default:"https://ntfy.sh"`
	NtfyToken string `config:"NTFY_TOKEN" default:""`

	ConfigDir string `config:"CONFIG_DIR" default:"/config"`

	secrets map[string]string
}

// New returns a HatsConfig for use by subscriber services.
// Overlay chain: defaults → env vars → HATS secrets (if HATS_HOST and HATS_TOKEN are set).
// If the HATS secrets fetch fails the error is returned but cfg is still usable with env-only values.
func New() (*HatsConfig, error) {
	cfg := &HatsConfig{}
	cfg.SetValues(ConfigValueSourceDefault).SetValues(ConfigValueSourceEnv)
	if err := cfg.applyHATSSecrets(); err != nil {
		return cfg, fmt.Errorf("HATS secrets overlay failed (continuing with env-only values): %w", err)
	}
	return cfg, nil
}

// applyHATSSecrets fetches all secrets from the HATS API and overlays them onto
// the config struct fields using config tag names as keys. It is a no-op if
// HatsHost or HatsToken is empty (i.e. when running as the HATS server itself).
func (c *HatsConfig) applyHATSSecrets() error {
	if c.HatsHost == "" || c.HatsToken == "" {
		return nil
	}
	hatsClient := client.NewHatsClient(c.GetHatsBaseUrl(), c.HatsToken)
	fetched, err := hatsClient.GetAllSecrets()
	if err != nil {
		return err
	}
	c.secrets = fetched
	c.SetValues(ConfigValueSourceHATS, fetched)
	return nil
}

type ConfigValueSource string

const ConfigValueSourceEnv ConfigValueSource = "env"
const ConfigValueSourceDefault ConfigValueSource = "default"
const ConfigValueSourceHATS ConfigValueSource = "hats"

// ApplyValues sets fields on v (must be a struct Value) using the given source.
// It handles struct fields with config tags via reflect.VisibleFields which
// promotes embedded struct fields.
func ApplyValues(v reflect.Value, source ConfigValueSource, inputs ...map[string]string) {
	fields := reflect.VisibleFields(v.Type())
	for _, field := range fields {
		envName := field.Tag.Get("config")
		if envName == "" {
			continue
		}
		var envValue string
		switch source {
		case ConfigValueSourceDefault:
			envValue = field.Tag.Get("default")
		case ConfigValueSourceEnv:
			envValue = os.Getenv(envName)
		case ConfigValueSourceHATS:
			if len(inputs) == 0 {
				break
			}
			if val, found := inputs[0][envName]; found {
				envValue = val
			}
		}
		if envValue == "" {
			continue
		}
		f := v.FieldByName(field.Name)
		if !f.IsValid() || !f.CanSet() {
			continue
		}
		switch field.Type.Kind() {
		case reflect.Bool:
			f.SetBool(strings.EqualFold(envValue, "true"))
		case reflect.Int:
			parsed, err := strconv.ParseInt(envValue, 10, 64)
			if err == nil {
				f.SetInt(parsed)
			}
		case reflect.String:
			f.SetString(envValue)
		}
	}
}

func (c *HatsConfig) SetValues(source ConfigValueSource, inputs ...map[string]string) *HatsConfig {
	ApplyValues(reflect.ValueOf(c).Elem(), source, inputs...)
	return c
}

func (c *HatsConfig) GetCustomSetting(name string, defaultValue string) string {
	if envValue := os.Getenv(name); envValue != "" {
		return envValue
	}
	if c.secrets != nil {
		if val, ok := c.secrets[name]; ok && val != "" {
			return val
		}
	}
	return defaultValue
}

func (c *HatsConfig) GetHomeAssistantBaseUrl() string {
	protocol := "http"
	if c.HomeAssistantSecure {
		protocol += "s"
	}
	return fmt.Sprintf("%s://%s:%s", protocol, c.HomeAssistantHost, c.HomeAssistantPort)
}

func (c *HatsConfig) GetHomeAssistantWebsocketUrl() string {
	protocol := "ws"
	if c.HomeAssistantSecure {
		protocol += "s"
	}
	return fmt.Sprintf("%s://%s:%s/api/websocket", protocol, c.HomeAssistantHost, c.HomeAssistantPort)
}

func (c *HatsConfig) GetHomeAssistantToken() string {
	return c.HomeAssistantToken
}

func (c *HatsConfig) GetNatsBaseUrl() string {
	return fmt.Sprintf("nats://%s:%s", c.NatsHost, c.NatsPort)
}

func (c *HatsConfig) GetHatsBaseUrl() string {
	if strings.HasPrefix(c.HatsHost, "http://") || strings.HasPrefix(c.HatsHost, "https://") {
		return c.HatsHost
	}
	protocol := "http"
	if c.HatsSecure {
		protocol += "s"
	}
	return fmt.Sprintf("%s://%s:%s", protocol, c.HatsHost, c.HatsPort)
}

func (c *HatsConfig) GetLogLevel() slog.Level {
	switch strings.ToLower(c.LogLvl) {
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

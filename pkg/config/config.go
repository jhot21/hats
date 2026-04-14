package config

import (
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strconv"
	"strings"

	"codeberg.org/jhot/hats/pkg/infisical"
)

type HatsConfig struct {
	LogLevl string `config:"LOG_LEVEL" default:"INFO"`

	InfisicalHost         string `config:"INFISICAL_HOST" default:"http://infisical:8080"`
	InfisicalClientID     string `config:"INFISICAL_CLIENT" default:""`
	InfisicalClientSecret string `config:"INFISICAL_SECRET" default:""`
	InfisicalProjectID    string `config:"INFISICAL_PROJECT" default:""`
	InfisicalEnvironment  string `config:"INFISICAL_ENVIRONMENT" default:"prod"`

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

	SyncthingHost  string `config:"SYNCTHING_HOST" default:"http://127.0.0.1:8384"`
	SyncthingToken string `config:"SYNCTHING_TOKEN" default:""`

	GokapiHost  string `config:"GOKAPI_HOST" default:"http://gokapi:53842"`
	GokapiToken string `config:"GOKAPI_TOKEN" default:""`

	ConfigDir string `config:"CONFIG_DIR" default:"/config"`

	infisicalClient *infisical.InfisicalClient
}

func New() (*HatsConfig, error) {
	cfg := &HatsConfig{}
	cfg.SetValues(ConfigValueSourceDefault).SetValues(ConfigValueSourceEnv)

	if cfg.InfisicalConfigured() {
		cfg.infisicalClient = infisical.New(cfg.InfisicalHost, cfg.InfisicalClientID, cfg.InfisicalClientSecret)

		err := cfg.infisicalClient.Login()
		if err != nil {
			cfg.infisicalClient = nil
			return cfg, fmt.Errorf("error logging in to Infisical: %w", err)
		}

		secrets, err := cfg.infisicalClient.ListSecrets(cfg.InfisicalProjectID, cfg.InfisicalEnvironment)
		if err != nil {
			return cfg, fmt.Errorf("error getting Infisical secrets: %w", err)
		}

		secretsMap := make(map[string]string)
		for _, secret := range secrets {
			secretsMap[secret.SecretKey] = secret.SecretValue
		}

		cfg.SetValues(ConfigValueSourceInfisical, secretsMap)
	}

	return cfg, nil
}

type ConfigValueSource string

const ConfigValueSourceEnv ConfigValueSource = "env"
const ConfigValueSourceDefault ConfigValueSource = "default"
const ConfigValueSourceInfisical ConfigValueSource = "infisical"

func (c *HatsConfig) SetValues(source ConfigValueSource, inputs ...map[string]string) *HatsConfig {
	fields := reflect.VisibleFields(reflect.TypeOf(*c))

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
		case ConfigValueSourceInfisical:
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

		f := reflect.ValueOf(c).Elem().FieldByName(field.Name)
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

	return c
}

func (c *HatsConfig) GetCustomSetting(name string, defaultValue string) string {
	returnValue := defaultValue

	envValue := os.Getenv(name)
	if envValue != "" {
		returnValue = envValue
	}

	if c.InfisicalConfigured() {
		secret, _ := c.infisicalClient.GetSecret(name, c.InfisicalProjectID, c.InfisicalEnvironment)
		if secret.SecretValue != "" {
			returnValue = secret.SecretValue
		}
	}

	return returnValue
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

func (c *HatsConfig) GetNatsBaseUrl() string {
	return fmt.Sprintf("nats://%s:%s", c.NatsHost, c.NatsPort)
}

func (c *HatsConfig) GetHatsBaseUrl() string {
	protocol := "http"
	if c.HatsSecure {
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

func (c *HatsConfig) InfisicalConfigured() bool {
	return c.InfisicalHost != "" &&
		c.InfisicalClientID != "" &&
		c.InfisicalClientSecret != "" &&
		c.InfisicalProjectID != "" &&
		c.InfisicalEnvironment != ""
}

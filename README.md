# HATS

[Home Assistant](https://www.home-assistant.io/) + [NATS](https://nats.io/) = HATS

## Features
- Push Home Assistant websocket events to a NATS message queue
- Caching proxy for Home Assistant API
- Generic implementations for subscribing to NATS topics
- Clients for some application APIs (limited functionality)
    - [Gokapi](https://github.com/Forceu/Gokapi)
    - [ntfy](https://github.com/binwiederhier/ntfy)
    - [qBittorrent](https://github.com/qbittorrent/qBittorrent)
    - [Syncthing](https://github.com/syncthing/syncthing)
    - [National Weather Service](https://www.weather.gov/)
		- [Amcrest Cameras](https://amcrest.com/)
		- [Infisical](https://infisical.com/)

## NATS Topics

- `homeassistant.states.{domain}.{entity}.{state}` - Home Assistant device state changes: [payload](https://www.home-assistant.io/docs/configuration/state_object/)
- `homeassistant.attributes.{domain}.{entity}.{state}` - When a device's attributes change but the state hasn't changed: [payload](https://www.home-assistant.io/docs/configuration/state_object/)
- `homeassistant.zha.{device IEEE}` - ZHA events: [payload](https://www.home-assistant.io/docs/configuration/state_object/)
- `homeassistant.zwave-scene.{device ID}` - ZwaveJS scene events: [payload](https://www.home-assistant.io/integrations/zwave_js/#scene-events-value-notification)
- `homeassistant.nfc.{tag ID}` - Home Assistant NFC tag scanned: [payload](https://www.home-assistant.io/docs/configuration/state_object/)
- `homeassistant.timer.{timer name}.finished` - Home Assistant timer finished: payload is simply `"finished"`
- `schedules.{schedule name}` - HATS schedule finished: payload is simply `"finished"`
- `command.{command name}` - Command called via HATS API: payload is a byte array of the HTTP Post body

## Example Client

```golang
package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"code.jhot.me/jhot/hats/pkg/client"
	"code.jhot.me/jhot/hats/pkg/config"
	ha "code.jhot.me/jhot/hats/pkg/homeassistant"
	"code.jhot.me/jhot/hats/pkg/nats"
	n "github.com/nats-io/nats.go"
)

var (
	logger     *slog.Logger
	hatsClient *client.HatsClient
	natsClient *nats.NatsConnection
)

func main() {
	cfg := config.FromEnvironment()
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.GetLogLevel(),
	}))

	hatsClient = client.NewHatsClient(cfg.GetHatsBaseUrl(), cfg.HatsToken)
	natsClient = nats.DefaultNatsConnection().WithJetstream(false).WithHostName(cfg.NatsHost).
		WithPort(cfg.NatsPort).WithConnectionOption(n.Name(cfg.NatsClientName))
	err := natsClient.Connect()
	if err != nil {
		panic(err)
	}
	defer natsClient.Close()

	go natsClient.GenericStateSubscriber(logger, "sun.sun", SunHandler)

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	<-sigch
	logger.Info("SIGTERM received")
}

func SunHandler(state ha.StateData) error {
	var service string
	if up := ha.StateToBool(state.State); up {
		service = ha.Services.TurnOff
	} else {
		service = ha.Services.TurnOn
	}
	return hatsClient.CallService("light.some_light", service)
}

```
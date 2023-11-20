# HATS

[Home Assistant](https://www.home-assistant.io/) + [NATS](https://nats.io/) = HATS

## Features
- Push Home Assistant websocket events to a NATS message queue
- Caching proxy for Home Assistant API
- Clients for some application APIs (limited functionality)
    - [Gokapi](https://github.com/Forceu/Gokapi)
    - [ntfy](https://github.com/binwiederhier/ntfy)
    - [qBittorrent](https://github.com/qbittorrent/qBittorrent)
    - [Syncthing](https://github.com/syncthing/syncthing)
    - [National Weather Service](https://www.weather.gov/)

## NATS Topics

- `homeassistant.states.{domain}.{entity}.{state}` - Home Assistant device state changes: [payload](https://www.home-assistant.io/docs/configuration/state_object/)
- `homeassistant.zha.{device IEEE}` - ZHA events: [payload](https://www.home-assistant.io/docs/configuration/state_object/)
- `homeassistant.nfc.{tag ID}` - Home Assistant NFC tag scanned: [payload](https://www.home-assistant.io/docs/configuration/state_object/)
- `homeassistant.timer.{timer name}.finished` - Home Assistant timer finished: payload is simply `"finished"`
- `schedules.{schedule name}` - HATS schedule finished: payload is simply `"finished"`
- `command.{command name}` - Command called via HATS API: payload is a byte array of the HTTP Post body

## Example Client

```golang
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"code.jhot.me/jhot/hats/pkg/client"
	"code.jhot.me/jhot/hats/pkg/config"
	ha "code.jhot.me/jhot/hats/pkg/homeassistant"
	n "code.jhot.me/jhot/hats/pkg/nats"
	"github.com/nats-io/nats.go"
)

var (
	logger     *slog.Logger
	hatsClient *client.HatsClient
	natsClient *n.NatsConnection
)

func main() {
	cfg := config.FromEnvironment()
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.GetLogLevel(),
	}))

	hatsClient = client.NewHatsClient(cfg.GetHatsBaseUrl(), cfg.HatsToken)
	natsClient = n.DefaultNatsConnection().WithJetstream(false).WithHostName(cfg.NatsHost).WithPort(cfg.NatsPort).WithConnectionOption(nats.Name(cfg.NatsClientName))
	defer natsClient.Close()

	go GenericStateListener("sun.sun", SunHandler)

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	<-sigch
	logger.Info("SIGTERM received")
}

func SunHandler(state ha.StateData) error {
	return hatsClient.CallService("light.some_light", ha.Services.TurnOn)
}

func GenericStateListener(entityId string, handler func(ha.StateData) error) {
	topic := fmt.Sprintf("homeassistant.states.%s.*", entityId)
	l := logger.With("topic", topic, "entity_id", entityId)
	l.Debug("Subscribing to topic")
	sub, ch, err := natsClient.Subscribe(topic)
	if err != nil {
		l.Error("Error subscribing to topic", "error", err)
		return
	}

	defer sub.Unsubscribe()

	for msg := range ch {
		msg.Ack()
		var data ha.EventData
		err = json.Unmarshal(msg.Data, &data)
		if err != nil {
			l.Error("Error parsing message", "error", err)
			continue
		}
		l.Debug("Event state " + data.NewState.State)
		err = handler(data.NewState)
		if err != nil {
			l.Error("Error handling state event", "error", err)
			continue
		}
	}
}

```
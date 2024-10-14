# HATS

[Home Assistant](https://www.home-assistant.io/) + [NATS](https://nats.io/) = HATS

I have been running this at home for over a year, but should be considered alpha software because breaking changes can occur at any time.

## Features
- Push Home Assistant websocket events to a NATS message queue
- Caching proxy for Home Assistant API
- Generic implementations for subscribing to NATS topics
- Clients for some APIs (limited functionality)
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

## Examples

See the example directory for an example Docker Compose and client implementation.
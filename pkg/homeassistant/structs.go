package homeassistant

import (
	"errors"
	"fmt"
	"time"
)

// Message types that can be returned from Home Assistants websocket API
var MessageType = struct {
	AuthRequired    string
	AuthOk          string
	AuthInvalid     string
	Result          string
	Event           string
	ZhaEvent        string
	ZwaveEvent      string
	SubscribeEvents string
	StateChanged    string
	TagScanned      string
	TimerStarted    string
	TimerFinished   string
}{
	AuthRequired:    "auth_required",
	AuthOk:          "auth_ok",
	AuthInvalid:     "auth_invalid",
	Result:          "result",
	Event:           "event",
	ZhaEvent:        "zha_event",
	ZwaveEvent:      "zwave_js_value_notification",
	SubscribeEvents: "subscribe_events",
	StateChanged:    "state_changed",
	TagScanned:      "tag_scanned",
	TimerStarted:    "timer.started",
	TimerFinished:   "timer.finished",
}

// Home Assistant device domains
var Domains = struct {
	Light         string
	Switch        string
	Lock          string
	Cover         string
	Homeassistant string
	Group         string
}{
	Light:         "light",
	Switch:        "switch",
	Lock:          "lock",
	Cover:         "cover",
	Homeassistant: "homeassistant",
	Group:         "group",
}

// Home Assistant services
var Services = struct {
	TurnOn           string
	TurnOff          string
	Toggle           string
	Reload           string
	Lock             string
	Unlock           string
	OpenCover        string
	CloseCover       string
	StopCover        string
	SelectOption     string
	SetHvacMode      string
	SetFanMode       string
	SetTemperature   string
	SetValue         string
	Start            string
	Change           string
	Cancel           string
	SetCoverPosition string
}{
	TurnOn:           "turn_on",
	TurnOff:          "turn_off",
	Toggle:           "toggle",
	Reload:           "reload",
	Lock:             "lock",
	Unlock:           "unlock",
	OpenCover:        "open_cover",
	CloseCover:       "close_cover",
	StopCover:        "stop_cover",
	SelectOption:     "select_option",
	SetHvacMode:      "set_hvac_mode",
	SetFanMode:       "set_fan_mode",
	SetTemperature:   "set_temperature",
	SetValue:         "set_value",
	Start:            "start",
	Change:           "change",
	Cancel:           "cancel",
	SetCoverPosition: "set_cover_position",
}

// Extra props that can be sent when calling a Home Assistant service
var ExtraProps = struct {
	Transition            string
	Brightness            string
	BrightnessPercent     string
	BrightnessPercentStep string
	HvacMode              string
	FanMode               string
	Temperature           string
	TargetTempHigh        string
	TargetTempLow         string
	Duration              string
	Value                 string
	Position              string
}{
	Transition:            "transition",
	Brightness:            "brightness",
	BrightnessPercent:     "brightness_pct",
	BrightnessPercentStep: "brightness_step_pct",
	HvacMode:              "hvac_mode",
	FanMode:               "fan_mode",
	Temperature:           "temperature",
	TargetTempHigh:        "target_temp_high",
	TargetTempLow:         "target_temp_low",
	Duration:              "duration",
	Value:                 "value",
	Position:              "position",
}

type ResultContext struct {
	Id string `json:"id,omitempty"`
}

type Result struct {
	Context ResultContext `json:"context,omitempty"`
}

type StateData struct {
	LastChanged string                 `json:"last_changed,omitempty"`
	LastUpdated string                 `json:"last_updated,omitempty"`
	State       string                 `json:"state,omitempty"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
	Context     interface{}            `json:"context,omitempty"`
}

func (data StateData) GetStringAttribute(name string) string {
	if val, exists := data.Attributes[name]; exists {
		if value, isString := val.(string); isString {
			return value
		}
	}

	return ""
}

func (data StateData) GetFriendlyName() string {
	return data.GetStringAttribute("friendly_name")
}

func (data StateData) GetLastChanged() (time.Time, error) {
	return time.Parse(time.RFC3339, data.LastChanged)
}

type EventData struct {
	EntityId string    `json:"entity_id,omitempty"`
	NewState StateData `json:"new_state,omitempty"`
	OldState StateData `json:"old_state,omitempty"`

	// ZHA
	DeviceIeee string      `json:"device_ieee,omitempty"`
	DeviceId   string      `json:"device_id,omitempty"`
	Command    string      `json:"command,omitempty"`
	Args       interface{} `json:"args,omitempty"`
	Params     interface{} `json:"params,omitempty"`

	// NFC
	TagId string `json:"tag_id,omitempty"`

	// ZwaveJS Scene
	Domain           string `json:"domain,omitempty"`
	NodeID           int    `json:"node_id,omitempty"`
	Endpoint         int    `json:"endpoint,omitempty"`
	CommandClass     int    `json:"command_class,omitempty"`
	CommandClassName string `json:"command_class_name,omitempty"`
	Label            string `json:"label,omitempty"`
	Property         string `json:"property,omitempty"`
	PropertyName     string `json:"property_name,omitempty"`
	PropertyKey      string `json:"property_key,omitempty"`
	PropertyKeyName  string `json:"property_key_name,omitempty"`
	Value            string `json:"value,omitempty"`
	ValueRaw         int    `json:"value_raw,omitempty"`
}

func (data EventData) GetDeviceFriendlyName() string {
	val := data.NewState.GetFriendlyName()

	if val == "" {
		return data.EntityId
	}

	return val
}

type Event struct {
	Data      EventData `json:"data,omitempty"`
	EventType string    `json:"event_type,omitempty"`
	TimeFired string    `json:"time_fired,omitempty"`
	Origin    string    `json:"origin,omitempty"`
}

type HassMessage struct {
	Type        string `json:"type"`
	Version     string `json:"ha_version,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
	Message     string `json:"message,omitempty"`
	Success     bool   `json:"success,omitempty"`
	Result      Result `json:"result,omitempty"`
	EventType   string `json:"event_type,omitempty"`
	Event       Event  `json:"event,omitempty"`
	Id          int    `json:"id,omitempty"`
}

type AuthMessage struct {
	Type        string `json:"type"`
	AccessToken string `json:"access_token,omitempty"`
}

type SubscribeEventsMessage struct {
	Type      string `json:"type"`
	EventType string `json:"event_type"`
	Id        int    `json:"id"`
}

type HistoryEntry struct {
	EntityId string `json:"entity_id,omitempty"`
	StateData
}

type HistoryResponse [][]HistoryEntry

func (r HistoryResponse) GetOnDuration(onState, offState string) (time.Duration, error) {
	var total time.Duration = 0
	if len(r) == 0 {
		return total, errors.New("empty history response")
	}

	entityHistory := r[0]
	var previousTime *time.Time

	for i, hist := range entityHistory {
		if hist.State == onState {
			previousTimeVal, err := hist.GetLastChanged()
			if err != nil {
				return total, fmt.Errorf("error parsing last changed timestamp for on-state: %w", err)
			}
			previousTime = &previousTimeVal
			continue
		}
		if i != 0 && previousTime != nil && hist.State == offState {
			currentTime, err := hist.GetLastChanged()
			if err != nil {
				return total, fmt.Errorf("error parsing last changed timestamp for off-state: %w", err)
			}
			total += currentTime.Sub(*previousTime)
		}
	}

	return total, nil
}

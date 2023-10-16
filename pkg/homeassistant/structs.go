package homeassistant

// Message types that can be returned from Home Assistants websocket API
var MessageType = struct {
	AuthRequired    string
	AuthOk          string
	AuthInvalid     string
	Result          string
	Event           string
	ZhaEvent        string
	SubscribeEvents string
	StateChanged    string
	TagScanned      string
}{
	AuthRequired:    "auth_required",
	AuthOk:          "auth_ok",
	AuthInvalid:     "auth_invalid",
	Result:          "result",
	Event:           "event",
	ZhaEvent:        "zha_event",
	SubscribeEvents: "subscribe_events",
	StateChanged:    "state_changed",
	TagScanned:      "tag_scanned",
}

// Home Assistant device domains
var Domains = struct {
	Light  string
	Switch string
	Lock   string
	Cover  string
}{
	Light:  "light",
	Switch: "switch",
	Lock:   "lock",
	Cover:  "cover",
}

// Home Assistant services
var Services = struct {
	TurnOn       string
	TurnOff      string
	Toggle       string
	Reload       string
	Lock         string
	Unlock       string
	OpenCover    string
	CloseCover   string
	SelectOption string
}{
	TurnOn:       "turn_on",
	TurnOff:      "turn_off",
	Toggle:       "toggle",
	Reload:       "reload",
	Lock:         "lock",
	Unlock:       "unlock",
	OpenCover:    "open_cover",
	CloseCover:   "close_cover",
	SelectOption: "select_option",
}

// Extra props that can be sent when calling a Home Assistant service
var ExtraProps = struct {
	Transition        string
	Brightness        string
	BrightnessPercent string
}{
	Transition:        "transition",
	Brightness:        "brightness",
	BrightnessPercent: "brightness_pct",
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

type EventData struct {
	EntityId   string      `json:"entity_id,omitempty"`
	NewState   StateData   `json:"new_state,omitempty"`
	OldState   StateData   `json:"old_state,omitempty"`
	DeviceIeee string      `json:"device_ieee,omitempty"`
	DeviceId   string      `json:"device_id,omitempty"`
	Command    string      `json:"command,omitempty"`
	Args       interface{} `json:"args,omitempty"`
	Params     interface{} `json:"params,omitempty"`
	TagId      string      `json:"tag_id,omitempty"`
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

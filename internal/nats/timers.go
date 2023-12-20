package nats

import (
	"fmt"
	"strings"
	"time"

	"code.jhot.me/jhot/hats/pkg/homeassistant"
)

var (
	haClient *homeassistant.RestClient
)

func initHomeAssistantClient() {
	if haClient == nil {
		haClient = homeassistant.NewRestClient(cfg.GetHomeAssistantBaseUrl(), cfg.HomeAssistantToken)
	}
}

type HatsTimer struct {
	Name       string
	State      string
	Duration   time.Duration
	FinishTime time.Time
}

func GetTimer(name string) (*HatsTimer, error) {
	initHomeAssistantClient()
	if !strings.HasPrefix(name, "timer.") {
		name = fmt.Sprintf("timer.%s", name)
	}
	state, err := haClient.GetState(name)
	if err != nil {
		return nil, err
	}

	d, err := homeassistant.ParseDuration(state.Attributes["duration"].(string))
	if err != nil {
		return nil, err
	}

	var finishTime time.Time
	if finishesAt, ok := state.Attributes["finishes_at"]; ok {
		finishTime, _ = time.Parse(time.RFC3339, finishesAt.(string))
	}

	return &HatsTimer{
		Name:       name,
		State:      state.State,
		Duration:   d,
		FinishTime: finishTime,
	}, nil
}

func (t *HatsTimer) Activate(durationOverride string) {
	d := t.Duration
	if durationOverride != "" {
		var err error
		d, err = time.ParseDuration(durationOverride)
		if err != nil {
			logger.Error("Error parsing duration", "error", err)
			d = t.Duration
		}
	}
	logger.Debug("Starting timer", "name", t.Name, "duration", int(d.Seconds()))
	err := haClient.CallService(t.Name, homeassistant.Services.Start, map[string]any{
		homeassistant.ExtraProps.Duration: int(d.Seconds()),
	})
	if err != nil {
		logger.Error("Error starting timer", "name", t.Name, "error", err)
	}
}

func (t *HatsTimer) ActivateIfNotAlready(durationOverride string) {
	if state, err := haClient.GetState(t.Name); err == nil && !homeassistant.StateToBool(state.State) {
		t.Activate(durationOverride)
	}
}

func (t *HatsTimer) Cancel() {
	haClient.CallService(t.Name, homeassistant.Services.Cancel)
}

func (t *HatsTimer) ToString() string {
	if homeassistant.StateToBool(t.State) {
		time, _ := t.FinishTime.MarshalText()
		return string(time)
	}
	return t.State
}

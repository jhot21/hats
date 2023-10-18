package nats

import (
	"fmt"
	"time"

	"code.jhot.me/jhot/hats/pkg/config"
	"code.jhot.me/jhot/hats/pkg/homeassistant"
)

var (
	haClient *homeassistant.RestClient
)

func initHomeAssistantClient() {
	if haClient == nil {
		cfg = config.FromEnvironment()
		haClient = homeassistant.NewRestClient(cfg.GetHomeAssistantBaseUrl(), cfg.HomeAssistantToken)
	}
}

type HatsTimer struct {
	Name       string
	State      string
	Duration   time.Duration
	FinishTime time.Time
}

func (t *HatsTimer) getEntityId() string {
	return fmt.Sprintf("timer.%s", t.Name)
}

func GetTimer(name string) (*HatsTimer, error) {
	initHomeAssistantClient()
	state, err := haClient.GetState(fmt.Sprintf("timer.%s", name))
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
	logger.Error("Starting timer", "duration", int(d.Seconds()))
	err := haClient.CallService(t.getEntityId(), homeassistant.Services.Start, map[string]any{
		homeassistant.ExtraProps.Duration: int(d.Seconds()),
	})
	if err != nil {
		logger.Error("Error starting timer", "error", err)
	}
}

func (t *HatsTimer) ActivateIfNotAlready(durationOverride string) {
	state, err := haClient.GetState(t.getEntityId())
	if err != nil || state.State != "active" {
		t.Activate(durationOverride)
	}
}

func (t *HatsTimer) Cancel() {
	haClient.CallService(t.getEntityId(), homeassistant.Services.Cancel)
}

func (t *HatsTimer) ToString() string {
	if t.State == "active" {
		time, _ := t.FinishTime.MarshalText()
		return string(time)
	}
	return t.State
}

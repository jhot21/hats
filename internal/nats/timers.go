package nats

import (
	"fmt"
	"strings"
	"sync"
	"time"

	ha "github.com/jhot21/go-ha-sdk"
)

var (
	haClient     *ha.Client
	haClientOnce sync.Once
)

func initHomeAssistantClient() {
	haClientOnce.Do(func() {
		haClient = ha.New(cfg.GetHomeAssistantBaseUrl(), cfg.HomeAssistantToken)
	})
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

	durStr, err := ha.CastAttribute[string](state, "duration")
	if err != nil {
		return nil, err
	}
	d, err := ha.ParseDuration(durStr)
	if err != nil {
		return nil, err
	}

	var finishTime time.Time
	if finishesAt, ok := state.Attributes["finishes_at"]; ok {
		if finishesAtStr, ok := finishesAt.(string); ok {
			finishTime, _ = time.Parse(time.RFC3339, finishesAtStr)
		}
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
	err := haClient.CallAction(t.Name, ha.ActionTimerStart, map[string]any{
		ha.PropDuration: int(d.Seconds()),
	})
	if err != nil {
		logger.Error("Error starting timer", "name", t.Name, "error", err)
	}
}

func (t *HatsTimer) ActivateIfNotAlready(durationOverride string) {
	if state, err := haClient.GetState(t.Name); err == nil && !ha.StateToBool(state) {
		t.Activate(durationOverride)
	}
}

func (t *HatsTimer) Cancel() {
	haClient.CallAction(t.Name, ha.ActionTimerCancel)
}

func (t *HatsTimer) ToString() string {
	if ha.StateToBool(ha.StateData{EntityID: t.Name, State: t.State}) {
		time, _ := t.FinishTime.MarshalText()
		return string(time)
	}
	return t.State
}

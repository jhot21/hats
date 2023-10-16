package nats

import (
	"errors"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

var (
	timerStore jetstream.KeyValue
	ticker     *time.Ticker
)

func TimerStoreConnect() error {
	if client.JS == nil {
		return errors.New("jetstream must be connected first")
	}

	logger.Debug("Looking for KV store")
	listener := client.JS.KeyValueStoreNames(ctx)
	found := false
	for name := range listener.Name() {
		if name == "KV_hats_timers" {
			found = true
		}
	}

	var err error
	if found {
		logger.Debug("Connecting to Timers KV store")
		timerStore, err = client.JS.KeyValue(ctx, "hats_timers")
	} else {
		logger.Debug("Creating Timers KV store")
		timerStore, err = client.JS.CreateKeyValue(ctx, jetstream.KeyValueConfig{
			Bucket: "hats_timers",
		})
	}

	return err
}

type HatsTimer struct {
	Name           string
	Duration       time.Duration
	NextActivation time.Time
}

func NewTimerWithDuration(name, duration string) *HatsTimer {
	t := &HatsTimer{
		Name: name,
	}

	d, err := time.ParseDuration(duration)
	if err != nil {
		d = 5 * time.Minute
	}
	t.Duration = d

	return t.CalculateNext()
}

func NewTimerWithActivation(name string, activation []byte) (*HatsTimer, error) {
	t := &HatsTimer{
		Name:     name,
		Duration: 5 * time.Minute,
	}

	a, err := time.Parse(time.RFC3339, string(activation))
	if err != nil {
		return t.CalculateNext(), err
	}
	t.NextActivation = a

	return t, nil
}

func GetTimer(name string) (*HatsTimer, error) {
	value, err := timerStore.Get(ctx, name)
	if err != nil {
		return nil, err
	}

	return NewTimerWithActivation(name, value.Value())
}

func (t *HatsTimer) CalculateNext() *HatsTimer {
	t.NextActivation = time.Now().Add(t.Duration)
	return t
}

func (t *HatsTimer) Marshall() []byte {
	timestamp, _ := t.NextActivation.MarshalText()
	return timestamp
}

func (t *HatsTimer) Activate() {
	timerStore.Put(ctx, t.Name, t.Marshall())
}

func (t *HatsTimer) ActivateIfNotAlready() {
	timerStore.Create(ctx, t.Name, t.Marshall())
}

func (t *HatsTimer) Cancel() {
	timerStore.Purge(ctx, t.Name)
}

func (t *HatsTimer) End() {
	t.Cancel()
	PublishString("timers."+t.Name, "done")
}

func WatchTimers() {
	ticker = time.NewTicker(time.Second)
	for {
		t := <-ticker.C
		timers, _ := timerStore.Keys(ctx, jetstream.IgnoreDeletes())
		for _, timerName := range timers {
			timer, err := GetTimer(timerName)
			if err != nil {
				logger.Error("Error retrieving timer", "timer", timerName, "error", err)
				timer.Cancel()
				continue
			}

			if t.After(timer.NextActivation) {
				timer.End()
			}
		}
	}
}

func StopTimers() {
	if ticker != nil {
		ticker.Stop()
	}
}

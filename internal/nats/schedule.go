package nats

import (
	"errors"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/nats-io/nats.go/jetstream"
)

var (
	scheduleStore jetstream.KeyValue
	scheduler     = gocron.NewScheduler(time.Local)
	schedules     = map[string]*gocron.Job{}

	fireSchedule = func(name string) {
		if err := PublishRequest("schedules."+name, []byte("fired"), 500*time.Millisecond, 3); err != nil {
			logger.Warn("Failed to fire schedule", "schedule", name, "error", err)
		}
	}
)

func ScheduleStoreConnect() error {
	if client.JS == nil {
		return errors.New("jetstream must be connected first")
	}

	var err error
	scheduleStore, err = client.JS.KeyValue(ctx, "hats_schedules")
	if errors.Is(err, jetstream.ErrBucketNotFound) {
		logger.Debug("Creating Schedules KV store")
		scheduleStore, err = client.JS.CreateKeyValue(ctx, jetstream.KeyValueConfig{
			Bucket: "hats_schedules",
		})
	}
	return err
}

func GetExistingSchedules() {
	scheduler.StartAsync()
	existing, _ := scheduleStore.Keys(ctx, jetstream.IgnoreDeletes())
	for _, name := range existing {
		sched, err := GetSchedule(name)
		if err != nil {
			continue
		}
		sched.Activate()
	}
}

type HatsSchedule struct {
	Name string
	Cron string
}

func NewSchedule(name, cron string) *HatsSchedule {
	return &HatsSchedule{
		Name: name,
		Cron: cron,
	}
}

func GetSchedule(name string) (*HatsSchedule, error) {
	value, err := scheduleStore.Get(ctx, name)
	if err != nil {
		return nil, err
	}

	return NewSchedule(name, string(value.Value())), nil
}

func (t *HatsSchedule) GetNext() string {
	if job, exists := schedules[t.Name]; exists {
		return job.NextRun().String()
	}
	return ""
}

func (t *HatsSchedule) Activate() error {
	job, err := scheduler.CronWithSeconds(t.Cron).Do(fireSchedule, t.Name)
	if err != nil {
		return err
	}
	if existing, found := schedules[t.Name]; found {
		scheduler.RemoveByID(existing)
	}
	schedules[t.Name] = job
	scheduleStore.PutString(ctx, t.Name, t.Cron)
	return nil
}

func (t *HatsSchedule) Cancel() {
	if job, exists := schedules[t.Name]; exists {
		scheduler.RemoveByID(job)
	}
	scheduleStore.Purge(ctx, t.Name)
}

func StopSchedules() {
	if scheduler != nil {
		scheduler.Stop()
	}
}

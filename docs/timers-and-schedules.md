# Timers and schedules

HATS provides two time-based triggering mechanisms: **timers** (countdown clocks) and **schedules** (cron expressions). Both fire NATS messages that your subscriber handles, and both are managed through the [HatsClient](hats-client-vs-ha-client.md).

---

## Timers

A HATS timer is a server-side countdown stored in JetStream. When it expires, HATS publishes `"finished"` to `homeassistant.timers.{name}.finished`.

Timers are **named** and **persistent across restarts** — if HATS restarts while a timer is running, the timer resumes from where it was.

### Managing timers

All timer operations go through [HatsClient](hats-client-vs-ha-client.md). The raw HTTP endpoints are documented in [api-reference.md](api-reference.md#timers).

```go
// Start a timer — duration is a Go duration string
// force=false: only starts if the timer isn't already running
// force=true: restarts the timer even if it is running
status, err := hatsClient.StartTimer("motion_off", "10m", false)

// Cancel a running timer
err := hatsClient.CancelTimer("motion_off")

// Get the current timer status (returns a status string)
status, err := hatsClient.GetTimer("motion_off")
```

### Subscribing to timer expiry

Use `SubscribeTimer` — the topic carries no useful payload, just the fact that it fired:

```go
natsClient.SubscribeTimer("motion_off", func() error {
    return hatsClient.CallService("light.living_room", homeassistant.Services.TurnOff)
})
```

See [subscriber-guide.md#subscription-methods](subscriber-guide.md#3-subscription-methods) for the full method reference.

### Motion timeout pattern

The most common timer use case: start a countdown when motion stops, turn off lights when the countdown expires.

```go
// When motion state changes
natsClient.SubscribeState("binary_sensor.living_room_motion",
    func(state homeassistant.StateData) error {
        if state.State == "on" {
            // Motion detected — cancel any pending off-timer
            hatsClient.CancelTimer("living_room_off")
            return hatsClient.CallService("light.living_room", homeassistant.Services.TurnOn)
        }
        // Motion stopped — start the countdown (won't restart if already running)
        _, err := hatsClient.StartTimer("living_room_off", "10m", false)
        return err
    })

// When timer expires — turn off lights
natsClient.SubscribeTimer("living_room_off", func() error {
    // Confirm motion is actually still off before acting
    if motionOn, _ := hatsClient.GetStateBool("binary_sensor.living_room_motion"); !motionOn {
        return hatsClient.CallService("light.living_room", homeassistant.Services.TurnOff)
    }
    return nil
})
```

---

## Schedules

A HATS schedule is a cron expression that fires a NATS message when it triggers. Unlike timers, schedules repeat automatically.

HATS uses **6-field cron syntax** with seconds as the first field:

```
┌─────────────── second (0–59)
│  ┌──────────── minute (0–59)
│  │  ┌───────── hour (0–23)
│  │  │  ┌────── day of month (1–31)
│  │  │  │  ┌─── month (1–12)
│  │  │  │  │  ┌─ day of week (0–6, Sun=0; or SUN, MON, ...)
│  │  │  │  │  │
0  *  *  *  *  *   → every minute, on the minute
0  0  *  *  *  *   → every hour
0  30 6  *  *  *   → 6:30 AM every day
0  0  8  *  *  MON → 8:00 AM every Monday
```

### Managing schedules

```go
// Create or update a schedule — idempotent, safe to call on every restart
_, err := hatsClient.SetSchedule("hourly", "0 0 * * * *")

// Delete a schedule
err := hatsClient.DeleteSchedule("hourly")

// Get next scheduled fire time
nextTime, err := hatsClient.GetSchedule("hourly")
```

**Always call `SetSchedule` before subscribing.** Schedules are stored in NATS JetStream, but if HATS loses its store or is freshly deployed, they need to be re-registered. Calling `SetSchedule` at subscriber startup is idempotent — it's safe to call every time.

### Subscribing to a schedule

```go
// Register the schedule first
hatsClient.SetSchedule("hourly", "0 0 * * * *")

// Then subscribe
natsClient.SubscribeSchedule("hourly", func() error {
    logger.Info("Hourly task running")
    return doHourlyWork()
})
```

### Multiple schedules

Register all schedules in one place before subscribing:

```go
// Set up all schedules
hatsClient.SetSchedule("minutely",      "0 * * * * *")
hatsClient.SetSchedule("hourly",        "0 0 * * * *")
hatsClient.SetSchedule("morning",       "0 0 7 * * *")
hatsClient.SetSchedule("night_weekday", "0 0 22 * * MON,TUE,WED,THU,FRI")
hatsClient.SetSchedule("night_weekend", "0 0 23 * * SAT,SUN")
hatsClient.SetSchedule("every_15min",   "0 */15 * * * *")

// Subscribe to each
natsClient.SubscribeSchedule("minutely",      minutelyHandler)
natsClient.SubscribeSchedule("hourly",        hourlyHandler)
natsClient.SubscribeSchedule("morning",       morningHandler)
natsClient.SubscribeSchedule("night_weekday", nightHandler)
natsClient.SubscribeSchedule("night_weekend", nightHandler)
natsClient.SubscribeSchedule("every_15min",   everyFifteenHandler)
```

### Seasonal schedules

Month and day-of-week fields let you target specific seasons or weekdays:

```go
// Summer peak hours: weekdays 2pm–7pm, May–September
hatsClient.SetSchedule("peak_start_summer", "0 0 14 * 5,6,7,8,9 MON,TUE,WED,THU,FRI")
hatsClient.SetSchedule("peak_end_summer",   "0 0 19 * 5,6,7,8,9 MON,TUE,WED,THU,FRI")

// Winter peak hours: weekdays 5pm–9pm, October–April
hatsClient.SetSchedule("peak_start_winter", "0 0 17 * 1,2,3,4,10,11,12 MON,TUE,WED,THU,FRI")
hatsClient.SetSchedule("peak_end_winter",   "0 0 21 * 1,2,3,4,10,11,12 MON,TUE,WED,THU,FRI")
```

---

← [Subscriber guide](subscriber-guide.md) · [Commands →](commands.md)

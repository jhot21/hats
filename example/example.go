package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"codeberg.org/jhot/hats/pkg/client"
	"codeberg.org/jhot/hats/pkg/config"
	"codeberg.org/jhot/hats/pkg/homeassistant"
	"codeberg.org/jhot/hats/pkg/nats"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		panic(err)
	}

	// Structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.GetLogLevel(),
	}))

	// Initialize HATS client
	hatsClient := client.NewHatsClient(cfg.GetHatsBaseUrl(), cfg.HatsToken)

	// Initialize NATS client and connect
	natsClient := nats.New(nats.WithoutJetstream,
		nats.WithHostName(cfg.NatsHost),
		nats.WithPort(cfg.NatsPort),
		nats.WithClientName(cfg.NatsClientName),
		nats.WithLogger(logger))
	err = natsClient.Connect()
	if err != nil {
		panic(err)
	}
	defer natsClient.Close()

	// Add a subscriber for state change event
	natsClient.SubscribeState("sun.sun", func(state homeassistant.StateData) error {
		if state.State == "above_horizon" {
			if motionOn, _ := hatsClient.GetStateBool("binary_sensor.living_room_motion"); !motionOn {
				logger.Info("Sun above horizon and motion off, turning off living room")
				return hatsClient.CallService("light.living_room", homeassistant.Services.TurnOff)
			}
		}

		return nil
	})

	// Wait for ctrl - c to end program
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	<-sigch
	logger.Info("SIGTERM received")
}

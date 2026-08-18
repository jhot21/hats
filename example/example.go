package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	ha "github.com/jhot21/go-ha-sdk"
	"github.com/jhot21/hats/pkg/client"
	"github.com/jhot21/hats/pkg/config"
	"github.com/jhot21/hats/pkg/nats"
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
	natsClient.SubscribeState("sun.sun", func(state ha.StateData) error {
		if state.State == "above_horizon" {
			if motionOn, _ := hatsClient.GetStateBool("binary_sensor.living_room_motion"); !motionOn {
				logger.Info("Sun above horizon and motion off, turning off living room")
				return hatsClient.CallService("light.living_room", string(ha.ActionTurnOff))
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

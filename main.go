package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"code.jhot.me/jhot/hats/internal/api"
	"code.jhot.me/jhot/hats/internal/homeassistant"
	"code.jhot.me/jhot/hats/internal/nats"
	"code.jhot.me/jhot/hats/pkg/config"
)

var (
	interrupt chan os.Signal
	cfg       *config.HatsConfig
	logger    *slog.Logger
	ctx       context.Context
	cancel    context.CancelFunc
)

func main() {
	cfg = config.FromEnvironment()
	ctx, cancel = context.WithCancel(context.Background())
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.GetLogLevel(),
	}))
	interrupt = make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	err := nats.JetstreamConnect(ctx, logger)
	if err != nil {
		panic(err)
	}
	defer nats.Close()

	err = nats.KvConnect()
	if err != nil {
		panic(err)
	}

	err = nats.ScheduleStoreConnect()
	if err != nil {
		panic(err)
	}
	nats.GetExistingSchedules()
	defer nats.StopSchedules()

	err = homeassistant.Subscribe(logger)
	if err != nil {
		panic(err)
	}
	defer homeassistant.CloseSubscription()

	api.Listen(logger)
	defer api.Close()

	for sig := range interrupt {
		logger.Debug("Interrupt:", "signal", sig.String())
		cancel()
		break
	}
}

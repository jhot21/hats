package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"codeberg.org/jhot/hats/internal/api"
	"codeberg.org/jhot/hats/internal/homeassistant"
	"codeberg.org/jhot/hats/internal/nats"
	"codeberg.org/jhot/hats/internal/ntfy"
	"codeberg.org/jhot/hats/internal/tokens"
	"codeberg.org/jhot/hats/pkg/config"
)

var (
	interrupt chan os.Signal
	cfg       *config.HatsConfig
	logger    *slog.Logger
	ctx       context.Context
	cancel    context.CancelFunc
)

func main() {
	var err error
	cfg, err = config.New()
	ctx, cancel = context.WithCancel(context.Background())
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.GetLogLevel(),
	}))

	if err != nil {
		logger.Error("Error during config initialization", "error", err)
	}

	interrupt = make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	err = nats.JetstreamConnect(ctx, logger, cfg)
	if err != nil {
		panic(err)
	}
	defer nats.Close()

	err = nats.KvConnect()
	if err != nil {
		panic(err)
	}

	ts, err := tokens.NewStore(ctx, logger, nats.JetStream())
	if err != nil {
		panic(err)
	}

	err = nats.ScheduleStoreConnect()
	if err != nil {
		panic(err)
	}
	nats.GetExistingSchedules()
	defer nats.StopSchedules()

	err = homeassistant.Subscribe(ctx, logger, cfg, cancel)
	if err != nil {
		panic(err)
	}
	defer homeassistant.CloseSubscription()

	ntfy.InitClient(cfg)

	api.InitAPI(logger, cfg, ts)
	defer api.Close()

	for sig := range interrupt {
		logger.Debug("Interrupt:", "signal", sig.String())
		cancel()
		break
	}
}

package main

import (
	"context"
	_ "embed"
	"log/slog"
	"os"
	"os/signal"

	"code.jhot.me/jhot/hats/internal/api"
	"code.jhot.me/jhot/hats/internal/homeassistant"
	"code.jhot.me/jhot/hats/internal/nats"
	"code.jhot.me/jhot/hats/internal/ntfy"
	"code.jhot.me/jhot/hats/pkg/config"
)

//go:embed README.md
var readme []byte

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

	err = nats.ScheduleStoreConnect()
	if err != nil {
		panic(err)
	}
	nats.GetExistingSchedules()
	defer nats.StopSchedules()

	err = homeassistant.Subscribe(logger, cfg)
	if err != nil {
		panic(err)
	}
	defer homeassistant.CloseSubscription()

	ntfy.InitClient(cfg)

	api.Listen(logger, cfg, readme)
	defer api.Close()

	for sig := range interrupt {
		logger.Debug("Interrupt:", "signal", sig.String())
		cancel()
		break
	}
}

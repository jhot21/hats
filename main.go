package main

import (
	"context"
	"encoding/hex"
	"log/slog"
	"os"
	"os/signal"

	"codeberg.org/jhot/hats/internal/admin"
	"codeberg.org/jhot/hats/internal/api"
	"codeberg.org/jhot/hats/internal/homeassistant"
	"codeberg.org/jhot/hats/internal/nats"
	"codeberg.org/jhot/hats/internal/ntfy"
	"codeberg.org/jhot/hats/internal/secrets"
	"codeberg.org/jhot/hats/internal/tokens"
	"codeberg.org/jhot/hats/pkg/config"
	"github.com/go-chi/chi/v5"
)

var (
	interrupt chan os.Signal
	cfg       *config.ServerConfig
	logger    *slog.Logger
	ctx       context.Context
	cancel    context.CancelFunc
)

func main() {
	var err error
	cfg, err = config.NewServer()
	ctx, cancel = context.WithCancel(context.Background())
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.GetLogLevel(),
	}))

	if err != nil {
		logger.Error("Error during config initialization", "error", err)
	}

	interrupt = make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	err = nats.JetstreamConnect(ctx, logger, &cfg.HatsConfig)
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

	var encKey []byte
	if cfg.SecretsEncryptionKey != "" {
		decoded, err := hex.DecodeString(cfg.SecretsEncryptionKey)
		if err != nil || len(decoded) != 32 {
			panic("SECRETS_ENCRYPTION_KEY must be a 64-char hex string (32 bytes); generate one with: openssl rand -hex 32")
		}
		encKey = decoded
	}

	ss, err := secrets.NewStore(ctx, nats.JetStream(), encKey, nats.Publish)
	if err != nil {
		panic(err)
	}

	err = nats.ScheduleStoreConnect()
	if err != nil {
		panic(err)
	}
	nats.GetExistingSchedules()
	defer nats.StopSchedules()

	err = homeassistant.Subscribe(ctx, logger, &cfg.HatsConfig, cancel)
	if err != nil {
		panic(err)
	}
	defer homeassistant.CloseSubscription()

	ntfy.InitClient(&cfg.HatsConfig)

	api.InitAPI(logger, &cfg.HatsConfig, func() chi.Router {
		router := chi.NewRouter()
		admin.Init(router, cfg, logger, ts, ss)
		return router
	}, ts, ss)
	defer api.Close()

	for sig := range interrupt {
		logger.Debug("Interrupt:", "signal", sig.String())
		cancel()
		break
	}
}

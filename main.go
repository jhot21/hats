package main

import (
	"context"
	"encoding/hex"
	"log/slog"
	"os"
	"os/signal"

	"codeberg.org/jhot/hats/internal/admin"
	"codeberg.org/jhot/hats/internal/api"
	"codeberg.org/jhot/hats/internal/cache"
	"codeberg.org/jhot/hats/internal/cmdstats"
	"codeberg.org/jhot/hats/internal/homeassistant"
	"codeberg.org/jhot/hats/internal/nats"
	"codeberg.org/jhot/hats/internal/ntfy"
	"codeberg.org/jhot/hats/internal/secrets"
	"codeberg.org/jhot/hats/internal/serverconfig"
	"codeberg.org/jhot/hats/internal/tokens"
	"github.com/go-chi/chi/v5"
)

var (
	interrupt chan os.Signal
	cfg       *serverconfig.ServerConfig
	logger    *slog.Logger
	ctx       context.Context
	cancel    context.CancelFunc
)

func main() {
	cfg = serverconfig.NewBootstrap()
	ctx, cancel = context.WithCancel(context.Background())
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.GetLogLevel(),
	}))

	interrupt = make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	err := nats.JetstreamConnect(ctx, logger, &cfg.HatsConfig)
	if err != nil {
		panic(err)
	}
	defer nats.Close()

	err = nats.KvConnect()
	if err != nil {
		panic(err)
	}

	err = nats.StatsKvConnect()
	if err != nil {
		panic(err)
	}

	cacheStats := cache.New(nats.StatsKV())
	if err := cacheStats.Load(ctx); err != nil {
		logger.Warn("Failed to load cache stats from KV", "error", err)
	}
	cacheStats.StartFlusher(ctx)

	err = nats.CmdStatsKvConnect()
	if err != nil {
		panic(err)
	}

	cmdStats := cmdstats.New(nats.CmdStatsKV(), logger)
	if err := cmdStats.Load(ctx); err != nil {
		logger.Warn("Failed to load command stats from KV", "error", err)
	}
	cmdStats.StartFlusher(ctx)

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

	// Phase 2: overlay config values stored in the secret store.
	if err := cfg.ApplySecretStore(ss); err != nil {
		logger.Warn("Secret store config overlay failed, using bootstrap values", "error", err)
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

	nc := ntfy.New(ss, cfg.NtfyHost, cfg.NtfyToken)

	mirror := cache.NewEntityMirror(nats.KV(), logger)
	mirror.Start(ctx)

	api.InitAPI(logger, &cfg.HatsConfig,
		func() chi.Router { return chi.NewRouter() },
		func(r chi.Router) { admin.Init(r, cfg, logger, ts, ss, cacheStats, mirror, cmdStats) },
		ts, ss, nc, cacheStats, cmdStats, mirror)
	defer api.Close()

	for sig := range interrupt {
		logger.Debug("Interrupt:", "signal", sig.String())
		logger.Info("Shutting down, flushing command stats...")
		cmdStats.Flush(context.Background())
		cancel()
		break
	}
}

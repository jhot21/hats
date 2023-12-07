package nats

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"code.jhot.me/jhot/hats/pkg/config"
	n "code.jhot.me/jhot/hats/pkg/nats"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

var (
	cfg    *config.HatsConfig
	client *n.NatsConnection
	kv     jetstream.KeyValue
	ctx    context.Context
	logger *slog.Logger
)

func Close() {
	client.Close()
}

func JetstreamConnect(parentContext context.Context, parentLogger *slog.Logger) error {
	ctx = parentContext
	logger = parentLogger
	cfg = config.FromEnvironment()
	var err error

	client = n.DefaultNatsConnection().WithHostName(cfg.NatsHost).WithPort(cfg.NatsPort).WithConnectionOption(nats.Name(cfg.NatsClientName))

	if cfg.NatsToken != "" {
		client.WithConnectionOption(nats.Token(cfg.NatsToken))
	}

	logger.Debug("Connecting to nats")
	err = client.Connect()
	if err != nil {
		return fmt.Errorf("%w: error connecting to nats server", err)
	}

	return nil
}

func KvConnect() error {
	if client.JS == nil {
		return errors.New("jetstream must be connected first")
	}

	logger.Debug("Looking for KV store")
	listener := client.JS.KeyValueStoreNames(ctx)
	found := false
	for name := range listener.Name() {
		if name == "KV_hats" {
			found = true
		}
	}

	var err error
	if found {
		logger.Debug("Connecting to KV store")
		kv, err = client.JS.KeyValue(ctx, "hats")
	} else {
		logger.Debug("Creating KV store")
		kv, err = client.JS.CreateKeyValue(ctx, jetstream.KeyValueConfig{
			Bucket: "hats",
			TTL:    2 * time.Hour,
		})
	}

	return err
}

func Publish(subject string, message []byte) {
	client.Publish(ctx, subject, message, jetstream.WithRetryAttempts(2), jetstream.WithRetryWait(500*time.Millisecond))
}

func PublishString(subject, message string) {
	Publish(subject, []byte(message))
}

func PublishRequest(subject string, message []byte, timeout time.Duration, retries int) {
	attempts := 0
	for {
		attempts += 1
		if attempts > retries {
			logger.Error("Request retries exceeded", "subject", subject)
			return
		}
		resp, err := client.Conn.Request(subject, message, timeout)

		if err == nil {
			logger.Debug("Request response received", "response", string(resp.Data), "attempts", attempts)
			return
		}
	}
}

func GetKeyValue(key string) ([]byte, error) {
	value, err := kv.Get(ctx, key)
	if err != nil {
		return []byte{}, err
	}
	return value.Value(), nil
}

func SetKeyValue(key string, value []byte) error {
	_, err := kv.Put(ctx, key, value)
	return err
}

func SetKeyValueString(key, value string) error {
	_, err := kv.PutString(ctx, key, value)
	return err
}

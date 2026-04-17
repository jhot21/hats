package nats

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"codeberg.org/jhot/hats/pkg/config"
	n "codeberg.org/jhot/hats/pkg/nats"
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

// IsConnected reports whether the NATS connection is currently active.
func IsConnected() bool {
	return client != nil && client.Conn != nil && client.Conn.IsConnected()
}

func JetstreamConnect(parentContext context.Context, parentLogger *slog.Logger, parentConfig *config.HatsConfig) error {
	ctx = parentContext
	logger = parentLogger
	cfg = parentConfig
	var err error

	client = n.New(n.WithHostName(cfg.NatsHost), n.WithPort(cfg.NatsPort), n.WithClientName(cfg.NatsClientName), n.WithToken(cfg.NatsToken))

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

	var err error
	logger.Debug("Connecting to KV store")
	kv, err = client.JS.KeyValue(ctx, "hats")
	if errors.Is(err, jetstream.ErrBucketNotFound) {
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

func PublishRequest(subject string, message []byte, timeout time.Duration, retries int) error {
	for attempt := 1; attempt <= retries; attempt++ {
		resp, err := client.Conn.Request(subject, message, timeout)
		if err == nil {
			logger.Debug("Request response received", "response", string(resp.Data), "attempts", attempt, "subject", subject)
			return nil
		}
		logger.Warn("Request failed", "subject", subject, "attempt", attempt, "error", err)
		if attempt < retries {
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
		}
	}
	logger.Warn("Request retries exceeded", "subject", subject)
	return errors.New("request retries exceeded")
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

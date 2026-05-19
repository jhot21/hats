package nats

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// watchBackoff is the starting backoff for RunWatch reconnect retries,
// stored as nanoseconds for atomic access. Overridden in tests for faster iteration.
var watchBackoff atomic.Int64

func init() {
	watchBackoff.Store(int64(time.Second))
}

// RunWatch starts a persistent KV watcher loop. It calls start() to open a
// watcher, passes every entry (including nil snapshot markers) to onEntry,
// and restarts with exponential backoff on channel close or start() failure.
// onRestart is called before each new watcher session (including the first)
// and may be nil. Exits when ctx is cancelled.
func RunWatch(
	ctx context.Context,
	logger *slog.Logger,
	name string,
	start func() (jetstream.KeyWatcher, error),
	onRestart func(),
	onEntry func(jetstream.KeyValueEntry),
) {
	backoff := time.Duration(watchBackoff.Load())
	for {
		if onRestart != nil {
			onRestart()
		}
		watcher, err := start()
		if err != nil {
			logger.Error(name+": failed to start watcher", "error", err, "backoff", backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
				backoff = min(backoff*2, 30*time.Second)
			}
			continue
		}
		backoff = time.Duration(watchBackoff.Load()) // reset after successful start
		if !runSession(ctx, watcher, onEntry) {
			return // ctx cancelled
		}
		logger.Warn(name+": watcher channel closed, restarting", "backoff", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
			backoff = min(backoff*2, 30*time.Second)
		}
	}
}

// runSession processes entries from watcher until ctx is cancelled or the
// channel closes. Returns true if channel closed (restart needed), false if
// ctx was cancelled.
func runSession(ctx context.Context, watcher jetstream.KeyWatcher, onEntry func(jetstream.KeyValueEntry)) bool {
	defer watcher.Stop()
	for {
		select {
		case <-ctx.Done():
			return false
		case entry, ok := <-watcher.Updates():
			if !ok {
				return true
			}
			onEntry(entry)
		}
	}
}

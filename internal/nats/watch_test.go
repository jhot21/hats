package nats

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// fakeWatcher satisfies jetstream.KeyWatcher using a plain channel.
// If the compiler reports missing interface methods, add no-op stubs below.
type fakeWatcher struct {
	ch chan jetstream.KeyValueEntry
}

func (f *fakeWatcher) Updates() <-chan jetstream.KeyValueEntry { return f.ch }
func (f *fakeWatcher) Stop() error                             { return nil }

func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// TestRunWatch_DeliversEntriesToCallback: entries and nil markers are forwarded.
func TestRunWatch_DeliversEntriesToCallback(t *testing.T) {
	watchBackoff.Store(int64(time.Millisecond))
	defer func() { watchBackoff.Store(int64(time.Second)) }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fw := &fakeWatcher{ch: make(chan jetstream.KeyValueEntry, 1)}
	fw.ch <- nil // nil = snapshot marker

	got := make(chan jetstream.KeyValueEntry, 1)
	go RunWatch(ctx, silentLogger(), "test",
		func() (jetstream.KeyWatcher, error) { return fw, nil },
		nil,
		func(e jetstream.KeyValueEntry) {
			got <- e
			cancel()
		},
	)

	select {
	case e := <-got:
		if e != nil {
			t.Errorf("expected nil entry, got %v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for entry")
	}
}

// TestRunWatch_RestartsOnChannelClose: closing the watcher channel triggers a restart.
func TestRunWatch_RestartsOnChannelClose(t *testing.T) {
	watchBackoff.Store(int64(time.Millisecond))
	defer func() { watchBackoff.Store(int64(time.Second)) }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	callCount := 0
	restarted := make(chan struct{})

	go RunWatch(ctx, silentLogger(), "test",
		func() (jetstream.KeyWatcher, error) {
			callCount++
			fw := &fakeWatcher{ch: make(chan jetstream.KeyValueEntry)}
			if callCount == 1 {
				close(fw.ch) // immediately trigger channel close
			} else {
				close(restarted)
				cancel()
			}
			return fw, nil
		},
		nil,
		func(e jetstream.KeyValueEntry) {},
	)

	select {
	case <-restarted:
	case <-time.After(time.Second):
		t.Fatal("timeout: watcher did not restart after channel close")
	}
}

// TestRunWatch_RetriesOnStartError: start() failures are retried until success.
func TestRunWatch_RetriesOnStartError(t *testing.T) {
	watchBackoff.Store(int64(time.Millisecond))
	defer func() { watchBackoff.Store(int64(time.Second)) }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	callCount := 0
	succeeded := make(chan struct{})

	go RunWatch(ctx, silentLogger(), "test",
		func() (jetstream.KeyWatcher, error) {
			callCount++
			if callCount < 3 {
				return nil, errors.New("connection refused")
			}
			close(succeeded)
			cancel()
			return &fakeWatcher{ch: make(chan jetstream.KeyValueEntry)}, nil
		},
		nil,
		func(e jetstream.KeyValueEntry) {},
	)

	select {
	case <-succeeded:
	case <-time.After(time.Second):
		t.Fatalf("timeout: start() called %d times, want 3", callCount)
	}
}

// TestRunWatch_ExitsOnCtxCancel: cancelling ctx stops RunWatch cleanly.
func TestRunWatch_ExitsOnCtxCancel(t *testing.T) {
	watchBackoff.Store(int64(time.Millisecond))
	defer func() { watchBackoff.Store(int64(time.Second)) }()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		RunWatch(ctx, silentLogger(), "test",
			func() (jetstream.KeyWatcher, error) {
				return &fakeWatcher{ch: make(chan jetstream.KeyValueEntry)}, nil
			},
			nil,
			func(e jetstream.KeyValueEntry) {},
		)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout: RunWatch did not exit after ctx cancel")
	}
}

// TestRunWatch_CallsOnRestart: onRestart is called before each new watcher session.
func TestRunWatch_CallsOnRestart(t *testing.T) {
	watchBackoff.Store(int64(time.Millisecond))
	defer func() { watchBackoff.Store(int64(time.Second)) }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sessionCount := 0
	restartCount := 0
	done := make(chan struct{})

	go func() {
		RunWatch(ctx, silentLogger(), "test",
			func() (jetstream.KeyWatcher, error) {
				sessionCount++
				fw := &fakeWatcher{ch: make(chan jetstream.KeyValueEntry)}
				if sessionCount >= 3 {
					cancel()
				} else {
					close(fw.ch) // trigger restart
				}
				return fw, nil
			},
			func() { restartCount++ },
			func(e jetstream.KeyValueEntry) {},
		)
		close(done)
	}()

	<-ctx.Done()
	<-done // wait for RunWatch goroutine to finish

	if restartCount < 3 {
		t.Errorf("expected onRestart called at least 3 times, got %d", restartCount)
	}
}

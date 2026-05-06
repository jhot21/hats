package tokens

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const bucketName = "hats_tokens"
const flushInterval = 60 * time.Second

var (
	ErrTokenNotFound = errors.New("token not found")
	ErrTokenExpired  = errors.New("token expired")
)

// TokenStore manages API tokens backed by NATS KV.
type TokenStore interface {
	Create(name string, scopes []string, expiresAt *time.Time) (rawToken string, err error)
	Validate(rawToken string) (*Token, error)
	Revoke(tokenHash string) error
	List() ([]*Token, error)
	Update(hash string, scopes []string, expiresAt *time.Time) error
}

type store struct {
	ctx    context.Context
	logger *slog.Logger
	kv     jetstream.KeyValue

	mu    sync.RWMutex
	cache map[string]*Token // hash → Token

	usageMu sync.Mutex
	usage   map[string]time.Time // hash → last used at
}

// NewStore creates a TokenStore connected to the hats_tokens NATS KV bucket.
// It starts a KV Watch goroutine and a flush goroutine immediately.
func NewStore(ctx context.Context, logger *slog.Logger, js jetstream.JetStream) (TokenStore, error) {
	kv, err := js.KeyValue(ctx, bucketName)
	if errors.Is(err, jetstream.ErrBucketNotFound) {
		kv, err = js.CreateKeyValue(ctx, jetstream.KeyValueConfig{
			Bucket: bucketName,
		})
	}
	if err != nil {
		return nil, err
	}

	s := &store{
		ctx:    ctx,
		logger: logger,
		kv:     kv,
		cache:  make(map[string]*Token),
		usage:  make(map[string]time.Time),
	}

	if err := s.startWatch(); err != nil {
		return nil, err
	}

	go s.flushLoop()
	return s, nil
}

func (s *store) startWatch() error {
	watcher, err := s.kv.WatchAll(s.ctx)
	if err != nil {
		return err
	}
	go s.runWatch(watcher)
	return nil
}

func (s *store) runWatch(watcher jetstream.KeyWatcher) {
	defer watcher.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case entry, ok := <-watcher.Updates():
			if !ok {
				// Channel closed — NATS connection dropped. Restart the watcher.
				s.logger.Warn("tokens: KV watcher closed, restarting")
				newWatcher, err := s.kv.WatchAll(s.ctx)
				if err != nil {
					s.logger.Error("tokens: failed to restart KV watcher", "error", err)
					return
				}
				watcher = newWatcher
				continue
			}
			if entry == nil {
				continue // signals end of initial values
			}
			hash := entry.Key()
			op := entry.Operation()
			if op == jetstream.KeyValueDelete || op == jetstream.KeyValuePurge {
				s.mu.Lock()
				delete(s.cache, hash)
				s.mu.Unlock()
				s.usageMu.Lock()
				delete(s.usage, hash)
				s.usageMu.Unlock()
				continue
			}
			var rec tokenRecord
			if err := json.Unmarshal(entry.Value(), &rec); err != nil {
				s.logger.Error("tokens: failed to unmarshal token record", "hash", hash, "error", err)
				continue
			}
			t := &Token{
				Hash:       hash,
				Name:       rec.Name,
				Scopes:     rec.Scopes,
				ExpiresAt:  rec.ExpiresAt,
				CreatedAt:  rec.CreatedAt,
				LastUsedAt: rec.LastUsedAt,
			}
			s.mu.Lock()
			s.cache[hash] = t
			s.mu.Unlock()
		}
	}
}

func (s *store) Create(name string, scopes []string, expiresAt *time.Time) (string, error) {
	rawToken, err := GenerateToken()
	if err != nil {
		return "", err
	}
	hash := HashToken(rawToken)
	rec := tokenRecord{
		Name:      name,
		Scopes:    scopes,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	if _, err := s.kv.Put(s.ctx, hash, data); err != nil {
		return "", err
	}
	return rawToken, nil
}

func (s *store) Validate(rawToken string) (*Token, error) {
	hash := HashToken(rawToken)
	s.mu.RLock()
	t, ok := s.cache[hash]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrTokenNotFound
	}
	if t.ExpiresAt != nil && time.Now().After(*t.ExpiresAt) {
		return nil, ErrTokenExpired
	}
	s.usageMu.Lock()
	s.usage[hash] = time.Now().UTC()
	s.usageMu.Unlock()
	return t, nil
}

func (s *store) Revoke(tokenHash string) error {
	// Clear in-memory immediately before the watch can propagate.
	s.mu.Lock()
	delete(s.cache, tokenHash)
	s.mu.Unlock()
	s.usageMu.Lock()
	delete(s.usage, tokenHash)
	s.usageMu.Unlock()

	err := s.kv.Delete(s.ctx, tokenHash)
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return nil
	}
	return err
}

// List returns pointers into the live cache. Callers must not mutate the returned Token values.
func (s *store) List() ([]*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Token, 0, len(s.cache))
	for _, t := range s.cache {
		result = append(result, t)
	}
	return result, nil
}

func (s *store) Update(hash string, scopes []string, expiresAt *time.Time) error {
	s.mu.RLock()
	old, ok := s.cache[hash]
	s.mu.RUnlock()
	if !ok {
		return ErrTokenNotFound
	}
	rec := tokenRecord{
		Name:       old.Name,
		Scopes:     scopes,
		ExpiresAt:  expiresAt,
		CreatedAt:  old.CreatedAt,
		LastUsedAt: old.LastUsedAt,
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	if _, err = s.kv.Put(s.ctx, hash, data); err != nil {
		return err
	}
	s.mu.Lock()
	if _, ok := s.cache[hash]; ok {
		s.cache[hash] = &Token{
			Hash:       hash,
			Name:       old.Name,
			Scopes:     scopes,
			ExpiresAt:  expiresAt,
			CreatedAt:  old.CreatedAt,
			LastUsedAt: old.LastUsedAt,
		}
	}
	s.mu.Unlock()
	return nil
}

func (s *store) flushLoop() {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.flush()
		}
	}
}

func (s *store) flush() {
	s.usageMu.Lock()
	snapshot := make(map[string]time.Time, len(s.usage))
	for k, v := range s.usage {
		snapshot[k] = v
	}
	s.usageMu.Unlock()

	for hash, lastUsed := range snapshot {
		s.mu.RLock()
		t, ok := s.cache[hash]
		s.mu.RUnlock()
		if !ok {
			continue // token was revoked between flush ticks
		}
		// Safe to read t's fields without holding mu: Token fields are set once at
		// construction in startWatch and never mutated afterward.
		lu := lastUsed
		rec := tokenRecord{
			Name:       t.Name,
			Scopes:     t.Scopes,
			ExpiresAt:  t.ExpiresAt,
			CreatedAt:  t.CreatedAt,
			LastUsedAt: &lu,
		}
		data, err := json.Marshal(rec)
		if err != nil {
			continue
		}
		s.kv.Put(s.ctx, hash, data) //nolint:errcheck
	}
}

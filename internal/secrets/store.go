package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	plainBucket = "hats_secrets"
	encBucket   = "hats_secrets_enc"
)

var nameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// SecretStore manages secrets backed by two NATS KV buckets.
type SecretStore interface {
	Get(name string) (*Secret, error)
	List() ([]*SecretMeta, error)
	GetAll() (map[string]string, error)
	Create(name, value string, typ SecretType) error
	Update(name, value string) error
	Delete(name string) error
}

type store struct {
	ctx     context.Context
	plain   jetstream.KeyValue
	enc     jetstream.KeyValue
	key     []byte
	publish func(subject string, data []byte)
}

// plainRecord is the JSON shape in hats_secrets.
type plainRecord struct {
	Type      SecretType `json:"type"`
	Value     string     `json:"value"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// encRecord is the JSON shape in hats_secrets_enc.
type encRecord struct {
	Nonce      []byte    `json:"nonce"`
	Ciphertext []byte    `json:"ciphertext"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewStore creates a SecretStore backed by NATS KV.
// encryptionKey must be 32 bytes (from SECRETS_ENCRYPTION_KEY hex) or nil.
// If nil and encrypted secrets exist in the bucket, NewStore returns an error.
func NewStore(ctx context.Context, js jetstream.JetStream, encryptionKey []byte, publish func(string, []byte)) (SecretStore, error) {
	plain, err := getOrCreateBucket(ctx, js, plainBucket)
	if err != nil {
		return nil, err
	}
	enc, err := getOrCreateBucket(ctx, js, encBucket)
	if err != nil {
		return nil, err
	}
	if encryptionKey == nil {
		keys, err := enc.Keys(ctx)
		if err == nil && len(keys) > 0 {
			return nil, errors.New("secrets: SECRETS_ENCRYPTION_KEY must be set — encrypted secrets exist in the store")
		}
	}
	return &store{ctx: ctx, plain: plain, enc: enc, key: encryptionKey, publish: publish}, nil
}

func getOrCreateBucket(ctx context.Context, js jetstream.JetStream, name string) (jetstream.KeyValue, error) {
	kv, err := js.KeyValue(ctx, name)
	if errors.Is(err, jetstream.ErrBucketNotFound) {
		kv, err = js.CreateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: name})
	}
	return kv, err
}

func (s *store) Get(name string) (*Secret, error) {
	entry, err := s.plain.Get(s.ctx, name)
	if err == nil {
		var rec plainRecord
		if err := json.Unmarshal(entry.Value(), &rec); err != nil {
			return nil, err
		}
		return &Secret{Name: name, Type: rec.Type, Value: rec.Value, CreatedAt: rec.CreatedAt, UpdatedAt: rec.UpdatedAt}, nil
	}
	if !errors.Is(err, jetstream.ErrKeyNotFound) {
		return nil, err
	}

	entry, err = s.enc.Get(s.ctx, name)
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return nil, ErrSecretNotFound
	}
	if err != nil {
		return nil, err
	}
	if s.key == nil {
		return nil, errors.New("secrets: SECRETS_ENCRYPTION_KEY not configured")
	}
	var rec encRecord
	if err := json.Unmarshal(entry.Value(), &rec); err != nil {
		return nil, err
	}
	plaintext, err := decrypt(s.key, rec.Nonce, rec.Ciphertext)
	if err != nil {
		return nil, err
	}
	return &Secret{Name: name, Type: TypeEncryptedString, Value: plaintext, CreatedAt: rec.CreatedAt, UpdatedAt: rec.UpdatedAt}, nil
}

func (s *store) List() ([]*SecretMeta, error) {
	var result []*SecretMeta

	keys, err := s.plain.Keys(s.ctx)
	if err != nil && !errors.Is(err, jetstream.ErrNoKeysFound) {
		return nil, err
	}
	for _, k := range keys {
		entry, err := s.plain.Get(s.ctx, k)
		if err != nil {
			continue
		}
		var rec plainRecord
		if err := json.Unmarshal(entry.Value(), &rec); err != nil {
			continue
		}
		result = append(result, &SecretMeta{Name: k, Type: rec.Type, CreatedAt: rec.CreatedAt, UpdatedAt: rec.UpdatedAt})
	}

	encKeys, err := s.enc.Keys(s.ctx)
	if err != nil && !errors.Is(err, jetstream.ErrNoKeysFound) {
		return nil, err
	}
	for _, k := range encKeys {
		entry, err := s.enc.Get(s.ctx, k)
		if err != nil {
			continue
		}
		var rec encRecord
		if err := json.Unmarshal(entry.Value(), &rec); err != nil {
			continue
		}
		result = append(result, &SecretMeta{Name: k, Type: TypeEncryptedString, CreatedAt: rec.CreatedAt, UpdatedAt: rec.UpdatedAt})
	}

	return result, nil
}

func (s *store) GetAll() (map[string]string, error) {
	result := make(map[string]string)

	keys, err := s.plain.Keys(s.ctx)
	if err != nil && !errors.Is(err, jetstream.ErrNoKeysFound) {
		return nil, err
	}
	for _, k := range keys {
		secret, err := s.Get(k)
		if err != nil {
			continue
		}
		result[k] = secret.Value
	}

	encKeys, err := s.enc.Keys(s.ctx)
	if err != nil && !errors.Is(err, jetstream.ErrNoKeysFound) {
		return nil, err
	}
	for _, k := range encKeys {
		secret, err := s.Get(k)
		if err != nil {
			continue
		}
		result[k] = secret.Value
	}

	return result, nil
}

func (s *store) Create(name, value string, typ SecretType) error {
	if !nameRegexp.MatchString(name) {
		return ErrInvalidName
	}
	if _, err := s.plain.Get(s.ctx, name); err == nil {
		return ErrNameConflict
	}
	if _, err := s.enc.Get(s.ctx, name); err == nil {
		return ErrNameConflict
	}

	now := time.Now().UTC()
	if typ == TypeEncryptedString {
		if s.key == nil {
			return errors.New("secrets: SECRETS_ENCRYPTION_KEY not configured")
		}
		nonce, ciphertext, err := encrypt(s.key, value)
		if err != nil {
			return err
		}
		data, err := json.Marshal(encRecord{Nonce: nonce, Ciphertext: ciphertext, CreatedAt: now, UpdatedAt: now})
		if err != nil {
			return err
		}
		if _, err := s.enc.Put(s.ctx, name, data); err != nil {
			return err
		}
	} else {
		data, err := json.Marshal(plainRecord{Type: typ, Value: value, CreatedAt: now, UpdatedAt: now})
		if err != nil {
			return err
		}
		if _, err := s.plain.Put(s.ctx, name, data); err != nil {
			return err
		}
	}

	s.publish(fmt.Sprintf("secret.%s.created", name), []byte("{}"))
	return nil
}

func (s *store) Update(name, value string) error {
	entry, err := s.plain.Get(s.ctx, name)
	if err == nil {
		var rec plainRecord
		if err := json.Unmarshal(entry.Value(), &rec); err != nil {
			return err
		}
		rec.Value = value
		rec.UpdatedAt = time.Now().UTC()
		data, err := json.Marshal(rec)
		if err != nil {
			return err
		}
		if _, err := s.plain.Put(s.ctx, name, data); err != nil {
			return err
		}
		s.publish(fmt.Sprintf("secret.%s.updated", name), []byte("{}"))
		return nil
	}
	if !errors.Is(err, jetstream.ErrKeyNotFound) {
		return err
	}

	entry, err = s.enc.Get(s.ctx, name)
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return ErrSecretNotFound
	}
	if err != nil {
		return err
	}
	if s.key == nil {
		return errors.New("secrets: SECRETS_ENCRYPTION_KEY not configured")
	}
	var rec encRecord
	if err := json.Unmarshal(entry.Value(), &rec); err != nil {
		return err
	}
	nonce, ciphertext, err := encrypt(s.key, value)
	if err != nil {
		return err
	}
	rec.Nonce = nonce
	rec.Ciphertext = ciphertext
	rec.UpdatedAt = time.Now().UTC()
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	if _, err := s.enc.Put(s.ctx, name, data); err != nil {
		return err
	}
	s.publish(fmt.Sprintf("secret.%s.updated", name), []byte("{}"))
	return nil
}

func (s *store) Delete(name string) error {
	_, err := s.plain.Get(s.ctx, name)
	if err == nil {
		if err := s.plain.Delete(s.ctx, name); err != nil {
			return err
		}
		s.publish(fmt.Sprintf("secret.%s.deleted", name), []byte("{}"))
		return nil
	}
	if !errors.Is(err, jetstream.ErrKeyNotFound) {
		return err
	}

	_, err = s.enc.Get(s.ctx, name)
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return ErrSecretNotFound
	}
	if err != nil {
		return err
	}
	if err := s.enc.Delete(s.ctx, name); err != nil {
		return err
	}
	s.publish(fmt.Sprintf("secret.%s.deleted", name), []byte("{}"))
	return nil
}

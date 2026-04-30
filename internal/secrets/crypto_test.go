package secrets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testKey() []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := testKey()
	nonce, ciphertext, err := encrypt(key, "hello world")
	require.NoError(t, err)
	plaintext, err := decrypt(key, nonce, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, "hello world", plaintext)
}

func TestDecrypt_WrongKey(t *testing.T) {
	key := testKey()
	nonce, ciphertext, err := encrypt(key, "secret")
	require.NoError(t, err)

	wrongKey := make([]byte, 32)
	_, err = decrypt(wrongKey, nonce, ciphertext)
	assert.Error(t, err)
}

func TestEncrypt_UniqueNonces(t *testing.T) {
	key := testKey()
	nonce1, _, err := encrypt(key, "value")
	require.NoError(t, err)
	nonce2, _, err := encrypt(key, "value")
	require.NoError(t, err)
	assert.NotEqual(t, nonce1, nonce2)
}

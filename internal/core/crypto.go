package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
)

var ErrInvalidKey = errors.New("nodra: invalid encryption key")

// KeyProvider supplies the current key for new writes and historical keys for reads. Implementations should keep key material outside persisted state and rotate by adding the new key before making it current.
type KeyProvider interface {
	Current() (id string, key []byte, err error)
	Get(id string) ([]byte, error)
}

// MemoryKeyProvider is a small concurrency-safe provider for tests and local applications. Production applications should implement KeyProvider over their OS keychain or secret manager.
type MemoryKeyProvider struct {
	mu      sync.RWMutex
	current string
	keys    map[string][]byte
}

func NewMemoryKeyProvider(id string, key []byte) (*MemoryKeyProvider, error) {
	p := &MemoryKeyProvider{keys: make(map[string][]byte)}
	if err := p.Add(id, key); err != nil {
		return nil, err
	}
	p.current = id
	return p, nil
}
func (p *MemoryKeyProvider) Add(id string, key []byte) error {
	if p == nil || id == "" {
		return ErrInvalidKey
	}
	if _, err := aes.NewCipher(key); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.keys == nil {
		p.keys = make(map[string][]byte)
	}
	p.keys[id] = append([]byte(nil), key...)
	return nil
}
func (p *MemoryKeyProvider) SetCurrent(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.keys[id]; !ok {
		return ErrInvalidKey
	}
	p.current = id
	return nil
}
func (p *MemoryKeyProvider) Current() (string, []byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	key, ok := p.keys[p.current]
	if !ok {
		return "", nil, ErrInvalidKey
	}
	return p.current, append([]byte(nil), key...), nil
}
func (p *MemoryKeyProvider) Get(id string) ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	key, ok := p.keys[id]
	if !ok {
		return nil, ErrInvalidKey
	}
	return append([]byte(nil), key...), nil
}

// AESGCMCodec encrypts an inner codec's bytes with a random nonce and stores only the key ID, nonce and ciphertext in a JSON envelope. It supports key rotation by reading historical IDs from the provider while new Encode calls use Current.
type AESGCMCodec[T any] struct {
	Inner Codec[T]
	Keys  KeyProvider
}

type encryptedEnvelope struct {
	Algorithm  string `json:"algorithm"`
	KeyID      string `json:"keyId"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

func (c AESGCMCodec[T]) Encode(value T) ([]byte, error) {
	if c.Inner == nil || c.Keys == nil {
		return nil, ErrInvalidPersistence
	}
	keyID, key, err := c.Keys.Current()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := c.Inner.Encode(value)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	envelope := encryptedEnvelope{Algorithm: "AES-GCM", KeyID: keyID, Nonce: base64.RawStdEncoding.EncodeToString(nonce), Ciphertext: base64.RawStdEncoding.EncodeToString(gcm.Seal(nil, nonce, plain, nil))}
	return json.Marshal(envelope)
}

func (c AESGCMCodec[T]) Decode(data []byte) (T, error) {
	var zero T
	if c.Inner == nil || c.Keys == nil {
		return zero, ErrInvalidPersistence
	}
	var envelope encryptedEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return zero, err
	}
	if envelope.Algorithm != "AES-GCM" || envelope.KeyID == "" {
		return zero, errors.New("nodra: invalid encrypted envelope")
	}
	key, err := c.Keys.Get(envelope.KeyID)
	if err != nil {
		return zero, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return zero, fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return zero, err
	}
	nonce, err := base64.RawStdEncoding.DecodeString(envelope.Nonce)
	if err != nil || len(nonce) != gcm.NonceSize() {
		return zero, errors.New("nodra: invalid encrypted nonce")
	}
	ciphertext, err := base64.RawStdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return zero, errors.New("nodra: invalid encrypted ciphertext")
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return zero, err
	}
	return c.Inner.Decode(plain)
}

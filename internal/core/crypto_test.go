package core

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestAESGCMCodecRoundTripAndRotation(t *testing.T) {
	keys, err := NewMemoryKeyProvider("v1", []byte("01234567890123456789012345678901"))
	requireNoError(t, err)
	codec := AESGCMCodec[string]{Inner: JSONCodec[string]{}, Keys: keys}
	encrypted, err := codec.Encode("secret state")
	requireNoError(t, err)
	if strings.Contains(string(encrypted), "secret state") {
		t.Fatal("plaintext leaked")
	}
	decoded, err := codec.Decode(encrypted)
	requireNoError(t, err)
	if decoded != "secret state" {
		t.Fatal(decoded)
	}
	requireNoError(t, keys.Add("v2", []byte("abcdefghijklmnopqrstuvwxyz123456")))
	requireNoError(t, keys.SetCurrent("v2"))
	rotated, err := codec.Encode("new state")
	requireNoError(t, err)
	if string(rotated) == string(encrypted) {
		t.Fatal("nonce/key rotation did not change envelope")
	}
	if value, err := codec.Decode(encrypted); err != nil || value != "secret state" {
		t.Fatalf("old key decode: %q %v", value, err)
	}
	if value, err := codec.Decode(rotated); err != nil || value != "new state" {
		t.Fatalf("new key decode: %q %v", value, err)
	}
}

func TestAESGCMCodecDetectsTamperingAndMissingKey(t *testing.T) {
	keys, err := NewMemoryKeyProvider("key", []byte("01234567890123456789012345678901"))
	requireNoError(t, err)
	codec := AESGCMCodec[int]{Inner: JSONCodec[int]{}, Keys: keys}
	data, err := codec.Encode(7)
	requireNoError(t, err)
	var envelope encryptedEnvelope
	requireNoError(t, json.Unmarshal(data, &envelope))
	ciphertext, err := base64.RawStdEncoding.DecodeString(envelope.Ciphertext)
	requireNoError(t, err)
	ciphertext[len(ciphertext)-1] ^= 1
	envelope.Ciphertext = base64.RawStdEncoding.EncodeToString(ciphertext)
	data, err = json.Marshal(envelope)
	requireNoError(t, err)
	if _, err := codec.Decode(data); err == nil {
		t.Fatal("tampering accepted")
	}
	other, err := NewMemoryKeyProvider("other", []byte("abcdefghijklmnopqrstuvwxyz123456"))
	requireNoError(t, err)
	if _, err := (AESGCMCodec[int]{Inner: JSONCodec[int]{}, Keys: other}).Decode(data); !errors.Is(err, ErrInvalidKey) && err == nil {
		t.Fatal("missing key accepted")
	}
	_ = context.Background()
}

func TestAESGCMCodecRejectsTypedNilDependencies(t *testing.T) {
	keys, err := NewMemoryKeyProvider("key", []byte("01234567890123456789012345678901"))
	requireNoError(t, err)
	var inner *jsonCodec[int]
	codec := AESGCMCodec[int]{Inner: inner, Keys: keys}
	if _, err := codec.Encode(1); !errors.Is(err, ErrInvalidPersistence) {
		t.Fatalf("typed-nil inner encode error = %v, want ErrInvalidPersistence", err)
	}
	if _, err := codec.Decode(nil); !errors.Is(err, ErrInvalidPersistence) {
		t.Fatalf("typed-nil inner decode error = %v, want ErrInvalidPersistence", err)
	}

	var typedNilKeys *MemoryKeyProvider
	codec = AESGCMCodec[int]{Inner: jsonCodec[int]{}, Keys: typedNilKeys}
	if _, err := codec.Encode(1); !errors.Is(err, ErrInvalidPersistence) {
		t.Fatalf("typed-nil keys encode error = %v, want ErrInvalidPersistence", err)
	}
	if _, err := codec.Decode(nil); !errors.Is(err, ErrInvalidPersistence) {
		t.Fatalf("typed-nil keys decode error = %v, want ErrInvalidPersistence", err)
	}
}

func TestMemoryKeyProviderNilReceiver(t *testing.T) {
	var provider *MemoryKeyProvider
	if err := provider.Add("key", []byte("01234567890123456789012345678901")); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("Add error = %v, want ErrInvalidKey", err)
	}
	if err := provider.SetCurrent("key"); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("SetCurrent error = %v, want ErrInvalidKey", err)
	}
	if _, _, err := provider.Current(); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("Current error = %v, want ErrInvalidKey", err)
	}
	if _, err := provider.Get("key"); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("Get error = %v, want ErrInvalidKey", err)
	}
}

type emptyIDKeyProvider struct{ key []byte }

func (p emptyIDKeyProvider) Current() (string, []byte, error) { return "", p.key, nil }
func (p emptyIDKeyProvider) Get(string) ([]byte, error)       { return p.key, nil }

func TestAESGCMCodecRejectsEmptyCurrentKeyID(t *testing.T) {
	codec := AESGCMCodec[int]{Inner: JSONCodec[int]{}, Keys: emptyIDKeyProvider{key: []byte("01234567890123456789012345678901")}}
	if _, err := codec.Encode(1); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("empty key ID error = %v, want ErrInvalidKey", err)
	}
}

// TestAESGCMCodecAssociatedDataBindsCiphertext checks that a ciphertext sealed with one AssociatedData opens only with that same value, so two stores sharing a key cannot load each other's files.
func TestAESGCMCodecAssociatedDataBindsCiphertext(t *testing.T) {
	keys, err := NewMemoryKeyProvider("v1", []byte("01234567890123456789012345678901"))
	requireNoError(t, err)
	settings := AESGCMCodec[string]{Inner: JSONCodec[string]{}, Keys: keys, AssociatedData: "app/settings.json"}
	data, err := settings.Encode("settings state")
	requireNoError(t, err)
	if value, err := settings.Decode(data); err != nil || value != "settings state" {
		t.Fatalf("round trip with associated data: %q %v", value, err)
	}
	if strings.Contains(string(data), "app/settings.json") {
		t.Fatal("associated data was written into the envelope")
	}
	session := AESGCMCodec[string]{Inner: JSONCodec[string]{}, Keys: keys, AssociatedData: "app/session.json"}
	if value, err := session.Decode(data); err == nil {
		t.Fatalf("ciphertext sealed for another associated data decoded as %q", value)
	}
	plain := AESGCMCodec[string]{Inner: JSONCodec[string]{}, Keys: keys}
	if value, err := plain.Decode(data); err == nil {
		t.Fatalf("ciphertext sealed with associated data decoded without it as %q", value)
	}
	unbound, err := plain.Encode("unbound state")
	requireNoError(t, err)
	if value, err := settings.Decode(unbound); err == nil {
		t.Fatalf("ciphertext sealed without associated data decoded with it as %q", value)
	}
}

// TestAESGCMCodecEmptyAssociatedDataKeepsFormat loads an envelope written before AssociatedData existed and checks that an empty value still seals with no associated data at all.
func TestAESGCMCodecEmptyAssociatedDataKeepsFormat(t *testing.T) {
	key := []byte("01234567890123456789012345678901")
	keys, err := NewMemoryKeyProvider("v1", key)
	requireNoError(t, err)
	codec := AESGCMCodec[string]{Inner: JSONCodec[string]{}, Keys: keys}
	// Written by AESGCMCodec before the AssociatedData field was added.
	saved := []byte(`{"algorithm":"AES-GCM","keyId":"v1","nonce":"DxFrw0zQIfBC0fFg","ciphertext":"RZHUkJu7e/MwTlUZ6u0PpmZew1AEfkUrLzAW+IrLV9CpAmMA7djIbgCpIpxWDQ"}`)
	if value, err := codec.Decode(saved); err != nil || value != "saved before associated data" {
		t.Fatalf("pre-existing envelope: %q %v", value, err)
	}

	data, err := codec.Encode("new state")
	requireNoError(t, err)
	var envelope encryptedEnvelope
	requireNoError(t, json.Unmarshal(data, &envelope))
	var fields map[string]json.RawMessage
	requireNoError(t, json.Unmarshal(data, &fields))
	if len(fields) != 4 {
		t.Fatalf("envelope fields = %v, want algorithm, keyId, nonce and ciphertext only", fields)
	}
	nonce, err := base64.RawStdEncoding.DecodeString(envelope.Nonce)
	requireNoError(t, err)
	ciphertext, err := base64.RawStdEncoding.DecodeString(envelope.Ciphertext)
	requireNoError(t, err)
	block, err := aes.NewCipher(key)
	requireNoError(t, err)
	gcm, err := cipher.NewGCM(block)
	requireNoError(t, err)
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	requireNoError(t, err)
	if string(plain) != `"new state"` {
		t.Fatalf("plaintext = %s", plain)
	}
}

// TestAESGCMCodecAssociatedDataWithKeyRotation checks that AssociatedData applies to every key of the provider: data sealed before and after a rotation opens only with the same value.
func TestAESGCMCodecAssociatedDataWithKeyRotation(t *testing.T) {
	keys, err := NewMemoryKeyProvider("v1", []byte("01234567890123456789012345678901"))
	requireNoError(t, err)
	codec := AESGCMCodec[string]{Inner: JSONCodec[string]{}, Keys: keys, AssociatedData: "app/settings.json"}
	old, err := codec.Encode("old state")
	requireNoError(t, err)
	requireNoError(t, keys.Add("v2", []byte("abcdefghijklmnopqrstuvwxyz123456")))
	requireNoError(t, keys.SetCurrent("v2"))
	rotated, err := codec.Encode("new state")
	requireNoError(t, err)
	var envelope encryptedEnvelope
	requireNoError(t, json.Unmarshal(rotated, &envelope))
	if envelope.KeyID != "v2" {
		t.Fatalf("rotated key ID = %q, want v2", envelope.KeyID)
	}
	if value, err := codec.Decode(old); err != nil || value != "old state" {
		t.Fatalf("old key decode: %q %v", value, err)
	}
	if value, err := codec.Decode(rotated); err != nil || value != "new state" {
		t.Fatalf("new key decode: %q %v", value, err)
	}
	other := AESGCMCodec[string]{Inner: JSONCodec[string]{}, Keys: keys, AssociatedData: "app/session.json"}
	for name, data := range map[string][]byte{"old key": old, "new key": rotated} {
		if value, err := other.Decode(data); err == nil {
			t.Fatalf("%s: ciphertext decoded with other associated data as %q", name, value)
		}
		if value, err := (AESGCMCodec[string]{Inner: JSONCodec[string]{}, Keys: keys}).Decode(data); err == nil {
			t.Fatalf("%s: ciphertext decoded without associated data as %q", name, value)
		}
	}
}

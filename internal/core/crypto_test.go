package core

import (
	"context"
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

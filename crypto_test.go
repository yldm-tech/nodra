package nodra

import (
	"context"
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
	data[len(data)-3] ^= 1
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

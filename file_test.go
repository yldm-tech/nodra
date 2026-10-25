package nodra

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFileStorageAndJSONCodecRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "tasks.json")
	storage := FileStorage{Path: path}
	store := New(counterState{Count: 4, Name: "ready"})
	p := NewPersistence(store, storage, JSONCodec[counterState]{})
	if err := p.Save(context.Background()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("perm=%v", info.Mode().Perm())
	}
	_ = store.Set(counterState{})
	if err := p.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.Get().Name != "ready" || store.Get().Count != 4 {
		t.Fatalf("value=%+v", store.Get())
	}
}
func TestFileStorageHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	storage := FileStorage{Path: filepath.Join(t.TempDir(), "state.json")}
	if err := storage.Save(ctx, []byte("x")); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
}

func TestFileStorageDurableRenameAndEncryptedPersistence(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory fsync is platform-specific")
	}
	path := filepath.Join(t.TempDir(), "durable", "state.json")
	keys, err := NewMemoryKeyProvider("v1", []byte("01234567890123456789012345678901"))
	requireNoError(t, err)
	store := New(42)
	p := NewPersistence(store, FileStorage{Path: path, Durable: true}, AESGCMCodec[int]{Inner: JSONCodec[int]{}, Keys: keys})
	requireNoError(t, p.Save(context.Background()))
	restored := New(0)
	requireNoError(t, NewPersistence(restored, FileStorage{Path: path, Durable: true}, AESGCMCodec[int]{Inner: JSONCodec[int]{}, Keys: keys}).Load(context.Background()))
	if restroredValue := restored.Get(); restroredValue != 42 {
		t.Fatalf("restored=%d", restroredValue)
	}
}

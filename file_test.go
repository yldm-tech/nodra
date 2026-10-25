package nodra

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

func TestJournaledFileStorageRoundTripAndCleanup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal", "state.json")
	storage := FileStorage{Path: path, Journal: true, Backup: true}
	if err := storage.Save(context.Background(), []byte(`{"count":4}`)); err != nil {
		t.Fatal(err)
	}
	data, err := storage.Load(context.Background())
	if err != nil || string(data) != `{"count":4}` {
		t.Fatalf("load=%q err=%v", data, err)
	}
	for _, suffix := range []string{".journal", ".commit"} {
		if _, err := os.Stat(path + suffix); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("journal artifact %s remains: %v", suffix, err)
		}
	}
}

func TestJournaledFileStorageRecoversCommittedRecord(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	oldData, newData := []byte(`{"count":1}`), []byte(`{"count":2}`)
	if err := os.WriteFile(path, oldData, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".journal", newData, 0600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(newData)
	if err := os.WriteFile(path+".commit", []byte(hex.EncodeToString(digest[:])), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := (FileStorage{Path: path, Journal: true}).Load(context.Background())
	if err != nil || string(data) != string(newData) {
		t.Fatalf("recovered=%q err=%v", data, err)
	}
	if _, err := os.Stat(path + ".commit"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("commit marker remains: %v", err)
	}
}

func TestJournaledFileStorageIgnoresUncommittedRecord(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	oldData, pending := []byte(`{"count":1}`), []byte(`{"count":2}`)
	if err := os.WriteFile(path, oldData, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".journal", pending, 0600); err != nil {
		t.Fatal(err)
	}
	data, err := (FileStorage{Path: path, Journal: true}).Load(context.Background())
	if err != nil || string(data) != string(oldData) {
		t.Fatalf("load=%q err=%v", data, err)
	}
}

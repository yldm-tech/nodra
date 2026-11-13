package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
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

func TestFileStorageAcceptsNilContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "state.json")
	storage := FileStorage{Path: path, Backup: true}
	var nilContext context.Context
	if err := storage.Save(nilContext, []byte("state")); err != nil {
		t.Fatalf("save error = %v", err)
	}
	data, err := storage.Load(nilContext)
	if err != nil || string(data) != "state" {
		t.Fatalf("load = %q, error = %v", data, err)
	}
	backup, err := storage.LoadBackup(nilContext)
	if err != nil || string(backup) != "state" {
		t.Fatalf("backup = %q, error = %v", backup, err)
	}
}

func TestFileStorageDurableRenameAndEncryptedPersistence(t *testing.T) {
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
	backup, err := storage.LoadBackup(context.Background())
	if err != nil || string(backup) != `{"count":4}` {
		t.Fatalf("journaled backup=%q err=%v", backup, err)
	}
	// Only the primary and its backup may remain: a journal, commit marker or temporary file left under any name means cleanup missed a record that saveJournaled wrote.
	requireDirectoryEntries(t, filepath.Dir(path), "state.json", "state.json.bak")
}

// The backup is written after the journal is cleaned up, so a failed backup reports that the primary was saved and leaves no journal record for a later Load to replay.
func TestJournaledFileStorageBackupFailureLeavesNoJournalRecord(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	// A non-empty directory where the backup belongs makes its atomic rename fail on every platform.
	requireNoError(t, os.MkdirAll(filepath.Join(path+".bak", "occupied"), 0755))
	storage := FileStorage{Path: path, Journal: true, Backup: true}
	err := storage.Save(context.Background(), []byte(`{"count":5}`))
	if err == nil || !strings.Contains(err.Error(), "primary saved, backup failed") {
		t.Fatalf("journaled save with a failing backup = %v", err)
	}
	requireDirectoryEntries(t, dir, "state.json", "state.json.bak")
	data, err := storage.Load(context.Background())
	if err != nil || string(data) != `{"count":5}` {
		t.Fatalf("load after failed backup=%q err=%v", data, err)
	}
}

func requireDirectoryEntries(t *testing.T, dir string, want ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	requireNoError(t, err)
	got := make([]string, 0, len(entries))
	for _, entry := range entries {
		got = append(got, entry.Name())
	}
	if !slices.Equal(got, want) {
		t.Fatalf("directory entries = %v, want %v", got, want)
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
	if _, err := os.Stat(path + ".journal"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("uncommitted journal remains: %v", err)
	}
}

// A crash between the two cleanup removals of a finished journaled save leaves a commit marker without its journal. The primary already holds the committed bytes, so recovery must drop the stale marker instead of failing every load.
func TestJournaledFileStorageDiscardsCommitMarkerWithoutJournal(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.json")
	storage := FileStorage{Path: path, Journal: true}
	saved := []byte(`{"count":2}`)
	requireNoError(t, storage.Save(ctx, saved))
	digest := sha256.Sum256(saved)
	requireNoError(t, os.WriteFile(path+".commit", []byte(hex.EncodeToString(digest[:])), 0600))
	for range 2 {
		data, err := storage.Load(ctx)
		if err != nil || string(data) != string(saved) {
			t.Fatalf("load=%q err=%v", data, err)
		}
	}
	if _, err := os.Stat(path + ".commit"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale commit marker remains: %v", err)
	}
	requireNoError(t, os.WriteFile(path+".commit", []byte(hex.EncodeToString(digest[:])), 0600))
	checked, err := storage.WithConflictDetection(ctx)
	if err != nil {
		t.Fatalf("conflict detection with stale marker: %v", err)
	}
	requireNoError(t, checked.Save(ctx, []byte(`{"count":3}`)))
	data, err := checked.Load(ctx)
	if err != nil || string(data) != `{"count":3}` {
		t.Fatalf("checked load=%q err=%v", data, err)
	}
}

// A commit marker from an earlier save next to the journal of a later save that stopped before committing must not wedge loads: the later journal was never committed, and the marker is only written after the primary is durable.
func TestJournaledFileStorageDiscardsStaleCommitMarkerWithUncommittedJournal(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.json")
	storage := FileStorage{Path: path, Journal: true}
	saved := []byte(`{"count":2}`)
	requireNoError(t, storage.Save(ctx, saved))
	digest := sha256.Sum256(saved)
	requireNoError(t, os.WriteFile(path+".commit", []byte(hex.EncodeToString(digest[:])), 0600))
	requireNoError(t, os.WriteFile(path+".journal", []byte(`{"count":3}`), 0600))
	data, err := storage.Load(ctx)
	if err != nil || string(data) != string(saved) {
		t.Fatalf("load=%q err=%v", data, err)
	}
	for _, suffix := range []string{".journal", ".commit"} {
		if _, err := os.Stat(path + suffix); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("journal artifact %s remains: %v", suffix, err)
		}
	}
}

// Durable must not turn every successful replacement into a failure on platforms without directory fsync, and the backup must still be written.
func TestFileStorageDurableSaveWritesBackup(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "durable", "state.json")
	storage := FileStorage{Path: path, Durable: true, Backup: true}
	requireNoError(t, storage.Save(ctx, []byte("state")))
	for _, candidate := range []string{path, path + ".bak"} {
		data, err := os.ReadFile(candidate)
		if err != nil || string(data) != "state" {
			t.Fatalf("%s=%q err=%v", candidate, data, err)
		}
	}
}

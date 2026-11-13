//go:build windows

package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// holdOpen opens path the way os.ReadFile and most scanners do, without delete sharing, so MoveFileEx cannot replace it until the handle closes.
func holdOpen(t *testing.T, path string) func() {
	t.Helper()
	r, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	var once sync.Once
	release := func() { once.Do(func() { _ = r.Close() }) }
	t.Cleanup(release)
	return release
}

func setReplaceRetryHook(t *testing.T, hook func(error)) {
	t.Helper()
	replaceRetryHook = hook
	t.Cleanup(func() { replaceRetryHook = nil })
}

func TestFileStorageSaveRetriesWhileReaderHoldsPrimaryWindows(t *testing.T) {
	for _, journal := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "state.json")
		storage := FileStorage{Path: path, Backup: true, Journal: journal}
		ctx := context.Background()
		if err := storage.Save(ctx, []byte("one")); err != nil {
			t.Fatalf("journal=%v first save: %v", journal, err)
		}
		release := holdOpen(t, path)
		retries := 0
		setReplaceRetryHook(t, func(err error) {
			retries++
			if !isReplaceContention(err) {
				t.Errorf("journal=%v retried %v, want only contention errors", journal, err)
			}
			// The reader goes away while Save is backing off, as a scan or another process's Load would.
			release()
		})
		if err := storage.Save(ctx, []byte("two")); err != nil {
			t.Fatalf("journal=%v save while a reader held the primary: %v", journal, err)
		}
		if retries == 0 {
			t.Fatalf("journal=%v: the replacement never met the open handle, so the test proves nothing", journal)
		}
		for _, p := range []string{path, path + ".bak"} {
			if got, err := os.ReadFile(p); err != nil || string(got) != "two" {
				t.Fatalf("journal=%v %s = %q, %v; want two", journal, filepath.Base(p), got, err)
			}
		}
	}
}

func TestFileStorageSaveStopsRetryingWhenContextEndsWindows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	storage := FileStorage{Path: path}
	if err := storage.Save(context.Background(), []byte("one")); err != nil {
		t.Fatal(err)
	}
	holdOpen(t, path)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setReplaceRetryHook(t, func(error) { cancel() })
	err := storage.Save(ctx, []byte("two"))
	if !errors.Is(err, context.Canceled) || !isReplaceContention(err) {
		t.Fatalf("Save = %v, want context.Canceled wrapped with the contention error", err)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != "one" {
		t.Fatalf("primary = %q, %v; want the previous state", got, err)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".nodra-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary files left behind: %v, %v", matches, err)
	}
}

func TestReplaceFileDoesNotRetryOtherErrorsWindows(t *testing.T) {
	dir := t.TempDir()
	setReplaceRetryHook(t, func(err error) { t.Errorf("retried %v", err) })
	err := replaceFile(context.Background(), filepath.Join(dir, "missing"), filepath.Join(dir, "state.json"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("replaceFile = %v, want a not-exist error without retries", err)
	}
}

func TestFileStorageReadOnlyPrimaryStaysReadOnlyWhenReplaceFailsWindows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	storage := FileStorage{Path: path, Perm: 0400}
	if err := storage.Save(context.Background(), []byte("one")); err != nil {
		t.Fatal(err)
	}
	holdOpen(t, path)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setReplaceRetryHook(t, func(error) { cancel() })
	if err := storage.Save(ctx, []byte("two")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Save = %v, want context.Canceled", err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0200 != 0 {
		t.Fatalf("primary mode = %v after a failed replacement, want the read-only attribute restored", info.Mode().Perm())
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != "one" {
		t.Fatalf("primary = %q, %v; want the previous state", got, err)
	}
}

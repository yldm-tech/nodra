package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// A Perm without the owner-write bit gives the file the read-only attribute on Windows, which MoveFileEx refuses to replace; rename on other systems ignores the destination's mode. Saving and recovering repeatedly must work on every system.
func TestFileStorageReadOnlyPermRepeatedSave(t *testing.T) {
	for _, journal := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "state.json")
		storage := FileStorage{Path: path, Perm: 0400, Backup: true, Journal: journal}
		ctx := context.Background()
		for _, data := range []string{"one", "two", "three"} {
			if err := storage.Save(ctx, []byte(data)); err != nil {
				t.Fatalf("journal=%v save %s: %v", journal, data, err)
			}
		}
		for _, p := range []string{path, path + ".bak"} {
			got, err := os.ReadFile(p)
			if err != nil || string(got) != "three" {
				t.Fatalf("journal=%v %s = %q, %v; want three", journal, filepath.Base(p), got, err)
			}
			info, err := os.Lstat(p)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm()&0200 != 0 {
				t.Fatalf("journal=%v %s mode = %v, want no owner-write bit", journal, filepath.Base(p), info.Mode().Perm())
			}
		}
		if !journal {
			continue
		}
		// A committed journal left by an interrupted save replaces the read-only primary on the next Load.
		digest := sha256.Sum256([]byte("four"))
		if err := os.WriteFile(storage.journalPath(), []byte("four"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(storage.commitPath(), []byte(hex.EncodeToString(digest[:])), 0600); err != nil {
			t.Fatal(err)
		}
		if got, err := storage.Load(ctx); err != nil || string(got) != "four" {
			t.Fatalf("recovering load = %q, %v; want four", got, err)
		}
	}
}

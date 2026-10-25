package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// JSONCodec encodes a store value using encoding/json.
type JSONCodec[T any] struct{}

func (JSONCodec[T]) Encode(value T) ([]byte, error) { return json.MarshalIndent(value, "", "  ") }
func (JSONCodec[T]) Decode(data []byte) (T, error) {
	var value T
	err := json.Unmarshal(data, &value)
	return value, err
}

// FileStorage stores encoded state at Path. Save writes a temporary file in
// the same directory and renames it into place, so readers see a complete file.
type FileStorage struct {
	Path string
	Perm os.FileMode
	// Backup writes an independent Path+".bak" copy after each successful primary replacement. A backup failure is returned even though the primary may already contain the new state. Use one Persistence writer per destination.
	Backup bool
	// Durable syncs the containing directory after each rename. This makes the file name update durable across a crash on filesystems that support directory fsync. It may add latency and is opt-in for compatibility.
	Durable bool
	// Journal enables a two-phase crash recovery record. The journal is committed
	// only after the primary rename and directory sync; an unfinished record is
	// discarded on the next Load. It adds extra filesystem writes and is opt-in.
	Journal bool
}

func (f FileStorage) LoadBackup(ctx context.Context) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !f.Backup || f.Path == "" {
		return nil, ErrNoBackup
	}
	return (FileStorage{Path: f.Path + ".bak"}).Load(ctx)
}

func (f FileStorage) Load(ctx context.Context) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.Path == "" {
		return nil, ErrInvalidPersistence
	}
	if f.Journal {
		if err := f.recoverJournal(ctx); err != nil {
			return nil, err
		}
	}
	return os.ReadFile(f.Path)
}
func (f FileStorage) Save(ctx context.Context, data []byte) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if f.Journal {
		return f.saveJournaled(ctx, data, nil)
	}
	return f.save(ctx, data, nil)
}

func (f FileStorage) journalPath() string { return f.Path + ".journal" }
func (f FileStorage) commitPath() string  { return f.Path + ".commit" }

func (f FileStorage) saveJournaled(ctx context.Context, data []byte, primarySaved func()) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if f.Path == "" {
		return ErrInvalidPersistence
	}
	dir := filepath.Dir(f.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	perm := f.Perm
	if perm == 0 {
		perm = 0600
	}
	// The journal is durable before the primary is replaced. A commit marker is
	// written only after the replacement and directory entry are durable.
	durable := runtime.GOOS != "windows"
	if err := writeAtomicFile(ctx, f.journalPath(), data, perm, durable); err != nil {
		return err
	}
	if err := writeAtomicFile(ctx, f.Path, data, perm, durable); err != nil {
		return fmt.Errorf("nodra: replace %s: %w", f.Path, err)
	}
	if primarySaved != nil {
		primarySaved()
	}
	digest := sha256.Sum256(data)
	if err := writeAtomicFile(ctx, f.commitPath(), []byte(hex.EncodeToString(digest[:])), 0600, durable); err != nil {
		return err
	}
	if err := os.Remove(f.journalPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Remove(f.commitPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if durable {
		if err := syncDirectory(dir); err != nil {
			return fmt.Errorf("nodra: sync %s after journal cleanup: %w", dir, err)
		}
	}
	if f.Backup {
		if err := (FileStorage{Path: f.Path + ".bak", Perm: f.Perm, Durable: f.Durable}).Save(ctx, data); err != nil {
			return fmt.Errorf("nodra: primary saved, backup failed: %w", err)
		}
	}
	return nil
}

func (f FileStorage) recoverJournal(ctx context.Context) error {
	if !f.Journal || f.Path == "" {
		return nil
	}
	marker, err := os.ReadFile(f.commitPath())
	if errors.Is(err, os.ErrNotExist) {
		// A journal without its commit marker was interrupted before the primary replacement became recoverable; discard the stale record so repeated loads do not retain an unfinished transaction forever.
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := os.Remove(f.journalPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}
	journal, err := os.ReadFile(f.journalPath())
	if err != nil {
		return fmt.Errorf("nodra: read committed journal: %w", err)
	}
	digest := sha256.Sum256(journal)
	if strings.TrimSpace(string(marker)) != hex.EncodeToString(digest[:]) {
		return errors.New("nodra: committed journal checksum mismatch")
	}
	perm := f.Perm
	if perm == 0 {
		perm = 0600
	}
	if err := writeAtomicFile(ctx, f.Path, journal, perm, runtime.GOOS != "windows"); err != nil {
		return fmt.Errorf("nodra: recover %s: %w", f.Path, err)
	}
	if err := os.Remove(f.journalPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Remove(f.commitPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if runtime.GOOS != "windows" {
		return syncDirectory(filepath.Dir(f.Path))
	}
	return nil
}

func writeAtomicFile(ctx context.Context, path string, data []byte, perm os.FileMode, durable bool) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".nodra-journal-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()
	if err = tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = os.Rename(name, path); err != nil {
		return err
	}
	if durable {
		return syncDirectory(dir)
	}
	return nil
}

func (f FileStorage) save(ctx context.Context, data []byte, primarySaved func()) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	if f.Path == "" {
		return ErrInvalidPersistence
	}
	dir := filepath.Dir(f.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".nodra-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()
	perm := f.Perm
	if perm == 0 {
		perm = 0600
	}
	if err = tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = os.Rename(name, f.Path); err != nil {
		return fmt.Errorf("nodra: replace %s: %w", f.Path, err)
	}
	if f.Durable {
		if err = syncDirectory(dir); err != nil {
			return fmt.Errorf("nodra: sync %s after replace: %w", dir, err)
		}
	}
	if primarySaved != nil {
		primarySaved()
	}
	if f.Backup {
		if err := (FileStorage{Path: f.Path + ".bak", Perm: f.Perm, Durable: f.Durable}).Save(ctx, data); err != nil {
			return fmt.Errorf("nodra: primary saved, backup failed: %w", err)
		}
	}
	return nil
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = directory.Close() }()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("directory sync: %w", err)
	}
	return nil
}

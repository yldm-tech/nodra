package nodra

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPersistenceExplicitBackupRecoveryPreservesOriginal(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "settings.json")
	store := New(7)
	storage := FileStorage{Path: path, Backup: true}
	p := NewPersistenceWithOptions(store, storage, JSONCodec[int]{}, PersistenceOptions[int]{Validate: func(v int) error {
		if v < 0 {
			return errors.New("negative")
		}
		return nil
	}})
	requireNoError(t, p.Save(ctx))
	corrupt := []byte(`{"incomplete":`)
	requireNoError(t, os.WriteFile(path, corrupt, 0600))
	requireNoError(t, store.Set(9))
	var loadErr *LoadError
	if err := p.Load(ctx); !errors.As(err, &loadErr) || loadErr.Stage != "decode" || loadErr.Backup {
		t.Fatalf("load: %v", err)
	}
	if store.Get() != 9 {
		t.Fatal("failed load changed state")
	}
	requireNoError(t, p.LoadBackup(ctx))
	if store.Get() != 7 {
		t.Fatal("backup not restored")
	}
	raw, err := os.ReadFile(path)
	requireNoError(t, err)
	if string(raw) != string(corrupt) {
		t.Fatal("recovery destroyed original")
	}
	requireNoError(t, os.WriteFile(path+".bak", []byte(`-1`), 0600))
	if err := p.LoadBackup(ctx); !errors.As(err, &loadErr) || loadErr.Stage != "validate" || !loadErr.Backup {
		t.Fatalf("backup validation: %v", err)
	}
	if store.Get() != 7 {
		t.Fatal("invalid backup changed state")
	}
	before, err := os.ReadFile(path + ".bak")
	requireNoError(t, err)
	requireNoError(t, store.Set(-2))
	if err := p.Save(ctx); err == nil {
		t.Fatal("invalid state saved")
	}
	after, err := os.ReadFile(path + ".bak")
	requireNoError(t, err)
	if string(before) != string(after) {
		t.Fatal("invalid save overwrote backup")
	}
}

func TestBackupSaveFailureAndMissingBackup(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.json")
	requireNoError(t, os.Mkdir(path+".bak", 0700))
	storage := FileStorage{Path: path, Backup: true}
	if err := storage.Save(ctx, []byte(`1`)); err == nil {
		t.Fatal("backup failure hidden")
	}
	raw, err := os.ReadFile(path)
	requireNoError(t, err)
	if string(raw) != "1" {
		t.Fatal("primary replacement missing")
	}
	p := NewPersistence(New(0), FileStorage{Path: path}, JSONCodec[int]{})
	if err := p.LoadBackup(ctx); !errors.Is(err, ErrNoBackup) {
		t.Fatalf("disabled backup: %v", err)
	}
}

func TestPersistenceLoadWithRecoveryFallsBackExplicitlyAndPreservesCauses(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "settings.json")
	storage := FileStorage{Path: path, Backup: true}
	store := New(7)
	p := NewPersistence(store, storage, JSONCodec[int]{})
	requireNoError(t, p.Save(ctx))
	requireNoError(t, os.WriteFile(path, []byte(`{"broken":`), 0600))
	requireNoError(t, store.Set(9))
	if err := p.LoadWithRecovery(ctx); err != nil {
		t.Fatal(err)
	}
	if store.Get() != 7 {
		t.Fatalf("recovered state = %d, want 7", store.Get())
	}
	if raw, err := os.ReadFile(path); err != nil || string(raw) != `{"broken":` {
		t.Fatalf("primary changed: %q, %v", raw, err)
	}

	requireNoError(t, os.WriteFile(path+".bak", []byte(`{"also":"broken"`), 0600))
	requireNoError(t, store.Set(11))
	var recoveryErr *RecoveryError
	if err := p.LoadWithRecovery(ctx); !errors.As(err, &recoveryErr) || !errors.Is(err, ErrPersistenceRecovery) {
		t.Fatalf("recovery error = %v", err)
	}
	if recoveryErr.Primary == nil || recoveryErr.Backup == nil || store.Get() != 11 {
		t.Fatalf("recovery causes/state = %+v, %d", recoveryErr, store.Get())
	}
}

func TestPersistenceLoadWithRecoveryDoesNotFallbackOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	storage := &recoveryProbeStorage{}
	p := NewPersistence(New(0), storage, JSONCodec[int]{})
	if err := p.LoadWithRecovery(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled recovery = %v", err)
	}
	if storage.backupLoads != 0 {
		t.Fatalf("canceled recovery touched backup %d times", storage.backupLoads)
	}
}

type recoveryProbeStorage struct{ backupLoads int }

func (*recoveryProbeStorage) Load(context.Context) ([]byte, error) {
	return nil, errors.New("primary unavailable")
}
func (s *recoveryProbeStorage) LoadBackup(context.Context) ([]byte, error) {
	s.backupLoads++
	return nil, errors.New("backup unavailable")
}
func (*recoveryProbeStorage) Save(context.Context, []byte) error { return nil }

func TestPersistenceLoadWithRecoveryKeepsConfigurationErrorsTyped(t *testing.T) {
	var p *Persistence[int]
	if err := p.LoadWithRecovery(context.Background()); !errors.Is(err, ErrInvalidPersistence) {
		t.Fatalf("nil persistence = %v", err)
	}
}

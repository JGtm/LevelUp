//go:build integration

// Package duckdb — sync_history_repo_integration_test.go : SyncHistoryRepo lit la clé de fin
// d'historique posée dans une player DB réelle (fichier), sans jamais créer la base absente.
package duckdb

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/migration"
)

func TestSyncHistoryRepo_BaseAbsenteNonCreee(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absente", "stats.duckdb")
	repo := NewSyncHistoryRepo(func(string, string) (string, error) { return path, nil }, "history_complete_at")

	got, err := repo.HistoryComplete(context.Background(), "jgtm", "halo_5")
	if err != nil || got {
		t.Fatalf("base absente : got=%v err=%v, want false/nil", got, err)
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Fatal("la lecture ne doit pas créer la player DB")
	}
}

func TestSyncHistoryRepo_HistoryComplete(t *testing.T) {
	ctx := context.Background()
	path := MigratedTempFile(t, migration.TargetPlayer, "stats.duckdb")
	repo := NewSyncHistoryRepo(func(string, string) (string, error) { return path, nil }, "history_complete_at")

	got, err := repo.HistoryComplete(ctx, "jgtm", "halo_5")
	if err != nil || got {
		t.Fatalf("clé absente : got=%v err=%v, want false/nil", got, err)
	}

	// L'état non complet est mémorisé une minute : l'horloge avance avant la relecture.
	now := time.Now()
	repo.now = func() time.Time { return now }
	now = now.Add(2 * syncHistoryRecheckEvery)

	// Handle RW tenu ouvert dans le process (comme le sync) : la lecture le réutilise.
	w, err := OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })
	if _, err := w.Exec(ctx, `INSERT INTO sync_meta (key, value, updated_at) VALUES ('history_complete_at', '2026-11-22T10:00:00Z', NOW())`); err != nil {
		t.Fatalf("seed sync_meta: %v", err)
	}
	got, err = repo.HistoryComplete(ctx, "jgtm", "halo_5")
	if err != nil || !got {
		t.Fatalf("clé posée : got=%v err=%v, want true/nil", got, err)
	}
}

func TestSyncHistoryRepo_PathError(t *testing.T) {
	boom := errors.New("joueur inconnu")
	repo := NewSyncHistoryRepo(func(string, string) (string, error) { return "", boom }, "history_complete_at")
	if _, err := repo.HistoryComplete(context.Background(), "x", "halo_5"); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want l'erreur du résolveur enveloppée", err)
	}
}

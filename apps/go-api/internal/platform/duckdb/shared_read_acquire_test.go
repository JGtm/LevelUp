// Package duckdb — shared_read_acquire_test.go : couverture du retry
// résilient d'acquisition de la connexion lecture shared (fix 2026-05-30 :
// la fenêtre de swap RO↔RW d'un sync concurrent faisait échouer en deadline
// la lecture progression → tables streaks/records/milestones vides).
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

// flakySharedReader échoue les `failuresBeforeSuccess` premiers Get puis réussit.
// Satisfait duckdb.SharedReader (Get(ctx) (*sql.DB, func(), error)).
type flakySharedReader struct {
	failuresBeforeSuccess int
	calls                 int
	releaseCalls          int
}

func (f *flakySharedReader) Get(_ context.Context) (*sql.DB, func(), error) {
	f.calls++
	if f.calls <= f.failuresBeforeSuccess {
		return nil, nil, errors.New("sharedprovider: swap timeout (simulated)")
	}
	return &sql.DB{}, func() { f.releaseCalls++ }, nil
}

func TestAcquireSharedRead_RetriesThenSucceeds(t *testing.T) {
	orig := sharedReadBackoff
	sharedReadBackoff = 2 * time.Millisecond
	defer func() { sharedReadBackoff = orig }()

	reader := &flakySharedReader{failuresBeforeSuccess: 2}
	db, release, err := AcquireSharedRead(context.Background(), reader)
	if err != nil {
		t.Fatalf("attendu succès après retries, got err: %v", err)
	}
	if db == nil {
		t.Fatal("db nil malgré succès")
	}
	if reader.calls != 3 {
		t.Errorf("tentatives: got %d, want 3 (2 échecs + 1 succès)", reader.calls)
	}
	release()
	if reader.releaseCalls != 1 {
		t.Errorf("release: got %d appels, want 1", reader.releaseCalls)
	}
}

func TestAcquireSharedRead_GivesUpAfterBudget(t *testing.T) {
	origBackoff := sharedReadBackoff
	origBudget := sharedReadBudget
	sharedReadBackoff = 2 * time.Millisecond
	sharedReadBudget = 25 * time.Millisecond
	defer func() {
		sharedReadBackoff = origBackoff
		sharedReadBudget = origBudget
	}()

	reader := &flakySharedReader{failuresBeforeSuccess: 1_000_000} // échoue toujours
	_, _, err := AcquireSharedRead(context.Background(), reader)
	if err == nil {
		t.Fatal("attendu erreur après dépassement du budget")
	}
	if reader.calls == 0 {
		t.Error("aucune tentative effectuée")
	}
}

// TestAcquireSharedRead_DetachesParentCancel : le ctx parent annulé
// ne doit pas tuer l'acquisition (post-sync best-effort en arrière-plan).
func TestAcquireSharedRead_DetachesParentCancel(t *testing.T) {
	orig := sharedReadBackoff
	sharedReadBackoff = 2 * time.Millisecond
	defer func() { sharedReadBackoff = orig }()

	parent, cancel := context.WithCancel(context.Background())
	cancel() // parent déjà annulé

	reader := &flakySharedReader{failuresBeforeSuccess: 1}
	db, release, err := AcquireSharedRead(parent, reader)
	if err != nil {
		t.Fatalf("le ctx parent annulé ne doit pas faire échouer (détaché), got: %v", err)
	}
	if db == nil {
		t.Fatal("db nil malgré succès")
	}
	release()
}

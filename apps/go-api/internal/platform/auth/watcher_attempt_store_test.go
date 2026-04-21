// Package auth — watcher_attempt_store_test.go : tests unitaires de WatcherAttemptStore.
package auth_test

import (
	"testing"

	auth "levelup/go-api/internal/platform/auth"
)

func TestWatcherAttemptStore_GetOrCreate_New(t *testing.T) {
	store := auth.NewWatcherAttemptStore()
	attempt, isNew := store.GetOrCreate()

	if attempt == nil {
		t.Fatal("expected attempt, got nil")
	}
	if !isNew {
		t.Error("expected isNew=true for first call")
	}
	if attempt.AttemptID == "" {
		t.Error("expected non-empty AttemptID")
	}
	if attempt.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", attempt.Status)
	}
}

func TestWatcherAttemptStore_GetOrCreate_Reuse(t *testing.T) {
	store := auth.NewWatcherAttemptStore()
	a1, _ := store.GetOrCreate()
	a2, isNew := store.GetOrCreate()

	if isNew {
		t.Error("expected isNew=false when pending attempt exists")
	}
	if a1.AttemptID != a2.AttemptID {
		t.Errorf("expected same AttemptID, got %q vs %q", a1.AttemptID, a2.AttemptID)
	}
}

func TestWatcherAttemptStore_GetOrCreate_AfterFailed(t *testing.T) {
	store := auth.NewWatcherAttemptStore()
	a1, _ := store.GetOrCreate()

	// Marquer la tentative comme échouée
	store.Update(a1.AttemptID, func(a *auth.WatcherAttempt) {
		a.Status = "failed"
	})

	// GetOrCreate doit créer une nouvelle tentative
	a2, isNew := store.GetOrCreate()
	if !isNew {
		t.Error("expected isNew=true after failed attempt")
	}
	if a2.AttemptID == a1.AttemptID {
		t.Error("expected different AttemptID after failed attempt")
	}
}

func TestWatcherAttemptStore_Snapshot_Found(t *testing.T) {
	store := auth.NewWatcherAttemptStore()
	attempt, _ := store.GetOrCreate()

	snap := store.Snapshot(attempt.AttemptID)
	if snap == nil {
		t.Fatal("expected snapshot, got nil")
	}
	if snap.AttemptID != attempt.AttemptID {
		t.Errorf("expected AttemptID %q, got %q", attempt.AttemptID, snap.AttemptID)
	}
}

func TestWatcherAttemptStore_Snapshot_NotFound(t *testing.T) {
	store := auth.NewWatcherAttemptStore()

	snap := store.Snapshot("nonexistent-id")
	if snap != nil {
		t.Error("expected nil for unknown AttemptID")
	}
}

func TestWatcherAttemptStore_Update_Applied(t *testing.T) {
	store := auth.NewWatcherAttemptStore()
	attempt, _ := store.GetOrCreate()

	store.Update(attempt.AttemptID, func(a *auth.WatcherAttempt) {
		a.Status = "authorized"
		a.Gamertag = "TestPlayer"
		a.UserCode = "ABCD-1234"
	})

	snap := store.Snapshot(attempt.AttemptID)
	if snap == nil {
		t.Fatal("expected snapshot after update")
	}
	if snap.Status != "authorized" {
		t.Errorf("expected status 'authorized', got %q", snap.Status)
	}
	if snap.Gamertag != "TestPlayer" {
		t.Errorf("expected gamertag 'TestPlayer', got %q", snap.Gamertag)
	}
}

func TestWatcherAttemptStore_Update_WrongID_NoOp(t *testing.T) {
	store := auth.NewWatcherAttemptStore()
	_, _ = store.GetOrCreate()

	// Update sur un ID inconnu ne doit pas paniquer
	store.Update("wrong-id", func(a *auth.WatcherAttempt) {
		a.Status = "authorized"
	})
}

func TestWatcherAttemptStore_Snapshot_IsCopy(t *testing.T) {
	store := auth.NewWatcherAttemptStore()
	attempt, _ := store.GetOrCreate()

	snap1 := store.Snapshot(attempt.AttemptID)
	snap1.Status = "authorized" // modifier la copie

	snap2 := store.Snapshot(attempt.AttemptID)
	if snap2.Status != "pending" {
		t.Error("Snapshot must return a copy; original should remain 'pending'")
	}
}

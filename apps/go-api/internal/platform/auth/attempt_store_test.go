// Package auth — attempt_store_test.go : tests unitaires de l'AttemptStore.
package auth_test

import (
	"testing"

	auth "levelup/go-api/internal/platform/auth"
)

func TestAttemptStore_GetOrCreate_New(t *testing.T) {
	store := auth.NewAttemptStore()
	attempt, isNew := store.GetOrCreate("session-1")

	if attempt == nil {
		t.Fatal("expected attempt, got nil")
	}
	if !isNew {
		t.Error("expected isNew=true for first call")
	}
	if attempt.SessionID != "session-1" {
		t.Errorf("expected SessionID 'session-1', got %q", attempt.SessionID)
	}
	if attempt.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", attempt.Status)
	}
}

func TestAttemptStore_GetOrCreate_Existing(t *testing.T) {
	store := auth.NewAttemptStore()
	a1, _ := store.GetOrCreate("session-1")
	a2, isNew := store.GetOrCreate("session-1")

	if isNew {
		t.Error("expected isNew=false for second call")
	}
	if a1.AttemptID != a2.AttemptID {
		t.Errorf("expected same attempt ID, got %q vs %q", a1.AttemptID, a2.AttemptID)
	}
}

func TestAttemptStore_Get_Found(t *testing.T) {
	store := auth.NewAttemptStore()
	attempt, _ := store.GetOrCreate("session-2")

	found := store.Get(attempt.AttemptID, "session-2")
	if found == nil {
		t.Fatal("expected attempt, got nil")
	}
}

func TestAttemptStore_Get_WrongSession(t *testing.T) {
	store := auth.NewAttemptStore()
	attempt, _ := store.GetOrCreate("session-3")

	// Tenter de récupérer avec un autre session_id → nil
	found := store.Get(attempt.AttemptID, "other-session")
	if found != nil {
		t.Fatal("expected nil for wrong session")
	}
}

func TestAttemptStore_Get_NotFound(t *testing.T) {
	store := auth.NewAttemptStore()
	found := store.Get("nonexistent-attempt", "any-session")
	if found != nil {
		t.Fatal("expected nil for unknown attempt")
	}
}

func TestAttemptStore_Update(t *testing.T) {
	store := auth.NewAttemptStore()
	attempt, _ := store.GetOrCreate("session-4")

	store.Update(attempt.AttemptID, func(a *auth.Attempt) {
		a.Status = "authorized"
		a.Gamertag = "TestPlayer"
	})

	snapshot := store.Snapshot(attempt.AttemptID)
	if snapshot == nil {
		t.Fatal("expected snapshot, got nil")
	}
	if snapshot.Status != "authorized" {
		t.Errorf("expected 'authorized', got %q", snapshot.Status)
	}
	if snapshot.Gamertag != "TestPlayer" {
		t.Errorf("expected gamertag 'TestPlayer', got %q", snapshot.Gamertag)
	}
}

func TestAttemptStore_Snapshot_NotFound(t *testing.T) {
	store := auth.NewAttemptStore()
	snap := store.Snapshot("nonexistent-id")
	if snap != nil {
		t.Fatal("expected nil for unknown attempt")
	}
}

func TestAttemptStore_DifferentSessions_Isolated(t *testing.T) {
	store := auth.NewAttemptStore()
	a1, _ := store.GetOrCreate("session-A")
	a2, _ := store.GetOrCreate("session-B")

	if a1.AttemptID == a2.AttemptID {
		t.Error("different sessions should have different attempt IDs")
	}
}

// Package auth — attempt_store_test.go : tests unitaires de l'AttemptStore.
package auth_test

import (
	"testing"
	"time"

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
		a.Status = auth.AttemptStatusAuthorized
		a.Gamertag = testGamertag
	})

	snapshot := store.Snapshot(attempt.AttemptID)
	if snapshot == nil {
		t.Fatal("expected snapshot, got nil")
	}
	if snapshot.Status != auth.AttemptStatusAuthorized {
		t.Errorf("expected 'authorized', got %q", snapshot.Status)
	}
	if snapshot.Gamertag != testGamertag {
		t.Errorf("expected gamertag %q, got %q", testGamertag, snapshot.Gamertag)
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

// TestAttemptStore_PurgeExpired vérifie que les tentatives plus vieilles que le
// TTL sont balayées (bornage mémoire — fix onboarding 2026-06-08).
func TestAttemptStore_PurgeExpired(t *testing.T) {
	store := auth.NewAttemptStoreWithTTL(20 * time.Millisecond)
	a, _ := store.GetOrCreate("session-ttl")

	// Avant expiration : la tentative est toujours là.
	if store.Snapshot(a.AttemptID) == nil {
		t.Fatal("tentative attendue avant expiration")
	}

	time.Sleep(40 * time.Millisecond)

	if n := store.PurgeExpired(); n != 1 {
		t.Errorf("PurgeExpired = %d, attendu 1", n)
	}
	if store.Snapshot(a.AttemptID) != nil {
		t.Error("tentative expirée doit être balayée")
	}
}

// TestAttemptStore_GetOrCreate_PurgesAndRecreates : après expiration, GetOrCreate
// purge en lazy et recrée une tentative neuve pour la même session.
func TestAttemptStore_GetOrCreate_PurgesAndRecreates(t *testing.T) {
	store := auth.NewAttemptStoreWithTTL(20 * time.Millisecond)
	a1, _ := store.GetOrCreate("session-recreate")

	time.Sleep(40 * time.Millisecond)

	a2, isNew := store.GetOrCreate("session-recreate")
	if !isNew {
		t.Error("attendu isNew=true après expiration de la tentative précédente")
	}
	if a1.AttemptID == a2.AttemptID {
		t.Error("une nouvelle tentative doit avoir un ID distinct après purge")
	}
}

// TestAttemptStore_PurgeKeepsFresh : une tentative récente n'est pas balayée.
func TestAttemptStore_PurgeKeepsFresh(t *testing.T) {
	store := auth.NewAttemptStoreWithTTL(time.Hour)
	a, _ := store.GetOrCreate("session-fresh")
	if n := store.PurgeExpired(); n != 0 {
		t.Errorf("PurgeExpired = %d, attendu 0 (tentative récente)", n)
	}
	if store.Snapshot(a.AttemptID) == nil {
		t.Error("tentative récente ne doit pas être balayée")
	}
}

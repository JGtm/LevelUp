package auth

import (
	"sync"
	"sync/atomic"
	"testing"
)

func authorizedAttempt(t *testing.T, s *AttemptStore, sessionID string) string {
	t.Helper()
	a, _, _ := s.GetOrCreate(sessionID, "")
	s.Update(a.AttemptID, func(a *Attempt) {
		a.Status = AttemptStatusAuthorized
		a.SpartanToken = "spartan"
		a.OAuthRefreshToken = "rt"
	})
	return a.AttemptID
}

func TestClaimAuthorized_OnceAndWipesSecrets(t *testing.T) {
	s := NewAttemptStore()
	id := authorizedAttempt(t, s, "sess-1")

	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if claimed, ok := s.ClaimAuthorized(id, "sess-1"); ok {
				wins.Add(1)
				if claimed.SpartanToken != "spartan" || claimed.OAuthRefreshToken != "rt" {
					t.Error("la copie remise doit porter les jetons")
				}
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("remises = %d, attendu 1", wins.Load())
	}
	snap := s.Snapshot(id)
	if snap.Status != AttemptStatusProvisioned {
		t.Errorf("statut = %q, attendu provisioned", snap.Status)
	}
	if snap.SpartanToken != "" || snap.OAuthRefreshToken != "" {
		t.Error("les jetons restent dans le registre après la remise")
	}
}

func TestClaimAuthorized_RefusesOtherSessionAndPending(t *testing.T) {
	s := NewAttemptStore()
	id := authorizedAttempt(t, s, "sess-1")
	if _, ok := s.ClaimAuthorized(id, "sess-2"); ok {
		t.Error("une autre session a reçu la tentative")
	}
	pending, _, _ := s.GetOrCreate("sess-3", "")
	if _, ok := s.ClaimAuthorized(pending.AttemptID, "sess-3"); ok {
		t.Error("une tentative en attente a été remise")
	}
}

func TestFail_SetsCodeAndWipesSecrets(t *testing.T) {
	s := NewAttemptStore()
	id := authorizedAttempt(t, s, "sess-1")
	s.Fail(id, "instance_locked")
	snap := s.Snapshot(id)
	if snap.Status != AttemptStatusFailed || snap.ErrorCode != "instance_locked" {
		t.Errorf("tentative = %q/%q, attendu failed/instance_locked", snap.Status, snap.ErrorCode)
	}
	if snap.SpartanToken != "" || snap.OAuthRefreshToken != "" {
		t.Error("les jetons restent dans le registre après l'échec")
	}
}

func TestRebind_FollowsNewSession(t *testing.T) {
	s := NewAttemptStore()
	id := authorizedAttempt(t, s, "old")
	s.Rebind(id, "new")
	if s.Get(id, "old") != nil {
		t.Error("l'ancienne session retrouve encore la tentative")
	}
	if s.Get(id, "new") == nil {
		t.Error("la nouvelle session ne retrouve pas la tentative")
	}
}

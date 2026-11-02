package auth

import (
	"errors"
	"testing"
	"time"
)

func TestAttemptStore_Browser_StateAUsageUnique(t *testing.T) {
	s := NewAttemptStore()
	a, err := s.StartBrowser("sess", "127.0.0.1", "state-1", "verifier-1", "fr")
	if err != nil {
		t.Fatalf("StartBrowser : %v", err)
	}
	claimed, err := s.ClaimBrowserState("state-1")
	if err != nil || claimed.CodeVerifier != "verifier-1" || claimed.AttemptID != a.AttemptID {
		t.Fatalf("ClaimBrowserState = %+v, %v", claimed, err)
	}
	if snap := s.Snapshot(a.AttemptID); snap.BrowserState != "" || snap.CodeVerifier != "" {
		t.Errorf("la poignée de main doit disparaître du registre : %+v", snap)
	}
	if _, err := s.ClaimBrowserState("state-1"); !errors.Is(err, ErrBrowserStateUnknown) {
		t.Errorf("rejeu = %v, attendu ErrBrowserStateUnknown", err)
	}
	if _, err := s.ClaimBrowserState(""); !errors.Is(err, ErrBrowserStateUnknown) {
		t.Errorf("state vide = %v, attendu ErrBrowserStateUnknown", err)
	}
}

func TestAttemptStore_Browser_ExpireApresLaDuree(t *testing.T) {
	s := NewAttemptStore()
	a, _ := s.StartBrowser("sess", "ip", "state-2", "v", "fr")
	s.Update(a.AttemptID, func(x *Attempt) { x.StartedAt = time.Now().Add(-BrowserStateTTL - time.Second) })
	if _, err := s.ClaimBrowserState("state-2"); !errors.Is(err, ErrBrowserStateExpired) {
		t.Fatalf("state expiré = %v", err)
	}
	if snap := s.Snapshot(a.AttemptID); snap.Status != AttemptStatusExpired {
		t.Errorf("statut = %q, attendu expired", snap.Status)
	}
	// Une tentative en attente périmée expire aussi au sondage, sans retour de Microsoft.
	b, _ := s.StartBrowser("sess-b", "ip", "state-3", "v", "fr")
	s.Update(b.AttemptID, func(x *Attempt) { x.StartedAt = time.Now().Add(-BrowserStateTTL - time.Second) })
	s.ExpireStaleBrowser(b.AttemptID)
	if snap := s.Snapshot(b.AttemptID); snap.Status != AttemptStatusExpired {
		t.Errorf("ExpireStaleBrowser : statut = %q, attendu expired", snap.Status)
	}
	if _, err := s.ClaimBrowserState("state-3"); !errors.Is(err, ErrBrowserStateUnknown) {
		t.Errorf("le state d'une tentative expirée est retiré : %v", err)
	}
}

func TestAttemptStore_Browser_NEstJamaisResserviAuCodeDAppareil(t *testing.T) {
	s := NewAttemptStore()
	b, _ := s.StartBrowser("sess", "ip", "state-4", "v", "fr")
	d, isNew, err := s.GetOrCreate("sess", "ip")
	if err != nil || !isNew || d.AttemptID == b.AttemptID {
		t.Fatalf("GetOrCreate a resservi la tentative du navigateur : %+v isNew=%v err=%v", d, isNew, err)
	}
	// La tentative du navigateur reste retrouvable par son identifiant.
	if s.Snapshot(b.AttemptID) == nil {
		t.Error("la tentative du navigateur doit survivre à un démarrage de code d'appareil")
	}
}

func TestAttemptStore_Browser_PlafondsCommunsAvecLeCodeDAppareil(t *testing.T) {
	s := NewAttemptStore()
	for i := 0; i < MaxPendingPerIP; i++ {
		if _, err := s.StartBrowser("sess-"+string(rune('a'+i)), "ip", "st-"+string(rune('a'+i)), "v", "fr"); err != nil {
			t.Fatalf("démarrage %d : %v", i, err)
		}
	}
	if _, err := s.StartBrowser("sess-z", "ip", "st-z", "v", "fr"); !errors.Is(err, ErrTooManyAttempts) {
		t.Errorf("au-delà du plafond par IP = %v, attendu ErrTooManyAttempts", err)
	}
}

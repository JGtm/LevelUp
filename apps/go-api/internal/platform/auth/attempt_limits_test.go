package auth

import (
	"errors"
	"fmt"
	"testing"
)

// La 21e tentative en attente (toutes IP confondues) est refusée ; une tentative
// déjà en attente pour la session reste servie (single-flight, hors plafond).
func TestAttemptStore_GlobalPendingCap(t *testing.T) {
	s := NewAttemptStore()
	for i := range MaxPendingAttempts {
		// IP distinctes : seul le plafond global peut refuser.
		if _, _, err := s.GetOrCreate(fmt.Sprintf("sess-%d", i), fmt.Sprintf("10.0.0.%d", i)); err != nil {
			t.Fatalf("tentative %d refusée: %v", i+1, err)
		}
	}
	_, _, err := s.GetOrCreate("sess-over", "10.9.9.9")
	if !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("21e tentative: err = %v, attendu ErrTooManyAttempts", err)
	}
	a, isNew, err := s.GetOrCreate("sess-0", "10.0.0.0")
	if err != nil || isNew || a == nil {
		t.Fatalf("la tentative en attente d'une session existante doit être resservie: a=%v isNew=%v err=%v", a, isNew, err)
	}
}

// Au plus MaxPendingPerIP tentatives en attente par IP cliente.
func TestAttemptStore_PerIPPendingCap(t *testing.T) {
	s := NewAttemptStore()
	for i := range MaxPendingPerIP {
		if _, _, err := s.GetOrCreate(fmt.Sprintf("sess-%d", i), "203.0.113.7"); err != nil {
			t.Fatalf("tentative %d refusée: %v", i+1, err)
		}
	}
	if _, _, err := s.GetOrCreate("sess-x", "203.0.113.7"); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("4e tentative de la même IP: err = %v, attendu ErrTooManyAttempts", err)
	}
	if _, _, err := s.GetOrCreate("sess-y", "198.51.100.1"); err != nil {
		t.Fatalf("une autre IP doit rester servie: %v", err)
	}
}

// Une tentative qui n'est plus en attente (échec, autorisation) libère sa place.
func TestAttemptStore_SettledAttemptFreesItsSlot(t *testing.T) {
	s := NewAttemptStore()
	var first string
	for i := range MaxPendingPerIP {
		a, _, err := s.GetOrCreate(fmt.Sprintf("sess-%d", i), "203.0.113.7")
		if err != nil {
			t.Fatalf("tentative %d refusée: %v", i+1, err)
		}
		if i == 0 {
			first = a.AttemptID
		}
	}
	s.Fail(first, "device_flow_acquire_error")
	if _, _, err := s.GetOrCreate("sess-new", "203.0.113.7"); err != nil {
		t.Fatalf("une place libérée doit être reprise: %v", err)
	}
}

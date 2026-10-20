package userstore

import (
	"errors"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
)

func newGuardStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "users.json"))
}

// TestLastAdmin_DeleteAndDemoteRefused : l'instance garde toujours un administrateur.
func TestLastAdmin_DeleteAndDemoteRefused(t *testing.T) {
	s := newGuardStore(t)
	if _, err := s.Create("boss", "boss-pass-1234", domain.RoleAdmin); err != nil {
		t.Fatalf("Create : %v", err)
	}
	if _, err := s.Create("bob", "bob-pass-12345", domain.RoleUser); err != nil {
		t.Fatalf("Create : %v", err)
	}
	if err := s.Delete("boss"); !errors.Is(err, ErrLastAdmin) {
		t.Errorf("Delete du dernier admin = %v, attendu ErrLastAdmin", err)
	}
	if err := s.SetRole("boss", domain.RoleUser); !errors.Is(err, ErrLastAdmin) {
		t.Errorf("rétrogradation du dernier admin = %v, attendu ErrLastAdmin", err)
	}
	if u, _ := s.Get("boss"); u == nil || u.Role != domain.RoleAdmin {
		t.Fatalf("le dernier admin a été modifié : %+v", u)
	}

	// Un second admin rend l'opération possible.
	if err := s.SetRole("bob", domain.RoleAdmin); err != nil {
		t.Fatalf("promotion : %v", err)
	}
	if err := s.SetRole("boss", domain.RoleUser); err != nil {
		t.Errorf("rétrogradation avec un autre admin : %v", err)
	}
	if err := s.Delete("boss"); err != nil {
		t.Errorf("suppression d'un non-admin : %v", err)
	}
}

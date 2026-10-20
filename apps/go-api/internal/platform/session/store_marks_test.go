// Package session — store_marks_test.go : les marques d'écriture de Touch (en mémoire) ne
// s'accumulent pas : oubliées à la suppression d'une session et à la purge des expirées.
package session

import (
	"path/filepath"
	"testing"
	"time"
)

func TestMarks_OublieesALaSuppressionEtALaPurge(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	s := NewStore(filepath.Join(t.TempDir(), "sessions"), time.Hour, "test-secret-32-bytesXXXXXXXXXX",
		WithClock(func() time.Time { return now }))
	a, b := s.New(), s.New()
	if err := s.Touch(a); err != nil {
		t.Fatalf("Touch a : %v", err)
	}
	if err := s.Touch(b); err != nil {
		t.Fatalf("Touch b : %v", err)
	}
	if len(s.marks) != 2 {
		t.Fatalf("marques = %d, attendu 2", len(s.marks))
	}

	if err := s.Delete(a.SessionID); err != nil {
		t.Fatalf("Delete : %v", err)
	}
	if _, ok := s.marks[a.SessionID]; ok {
		t.Error("la marque d'une session supprimée doit être oubliée")
	}

	now = now.Add(2 * time.Hour) // b dépasse le TTL d'une heure
	if removed := s.PurgeExpired(); removed != 1 {
		t.Errorf("PurgeExpired = %d, attendu 1", removed)
	}
	if len(s.marks) != 0 {
		t.Errorf("marques après purge = %d, attendu 0", len(s.marks))
	}
}

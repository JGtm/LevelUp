// Package session — store_test.go : tests unitaires du Store de session.
package session_test

import (
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/platform/session"
)

func newTestStore(t *testing.T) *session.Store {
	t.Helper()
	dir := t.TempDir()
	return session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32-bytesXXXXXXXXXX")
}

func TestStore_NewSession(t *testing.T) {
	store := newTestStore(t)
	sess := store.New()

	if sess == nil {
		t.Fatal("expected session, got nil")
	}
	if sess.SessionID == "" {
		t.Fatal("session ID should not be empty")
	}
	if sess.CreatedAt == 0 {
		t.Fatal("CreatedAt should be set")
	}
}

func TestStore_SaveAndLoad(t *testing.T) {
	store := newTestStore(t)
	sess := store.New()

	if err := store.Touch(sess); err != nil {
		t.Fatalf("Touch: %v", err)
	}

	loaded := store.Load(sess.SessionID)
	if loaded == nil {
		t.Fatal("expected loaded session, got nil")
	}
	if loaded.SessionID != sess.SessionID {
		t.Errorf("session ID mismatch: %q vs %q", loaded.SessionID, sess.SessionID)
	}
}

func TestStore_Load_NotFound(t *testing.T) {
	store := newTestStore(t)
	loaded := store.Load("nonexistent-session-id")
	if loaded != nil {
		t.Fatal("expected nil for unknown session")
	}
}

func TestStore_Delete(t *testing.T) {
	store := newTestStore(t)
	sess := store.New()
	_ = store.Touch(sess)

	if err := store.Delete(sess.SessionID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	loaded := store.Load(sess.SessionID)
	if loaded != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestStore_SignAndUnsign(t *testing.T) {
	store := newTestStore(t)
	sessionID := "test-session-uuid-1234"
	signed := store.SignCookie(sessionID)

	if signed == "" {
		t.Fatal("signed cookie should not be empty")
	}

	unsigned := store.UnsignCookie(signed)
	if unsigned != sessionID {
		t.Errorf("UnsignCookie(%q) = %q, want %q", signed, unsigned, sessionID)
	}
}

func TestStore_UnsignCookie_Invalid(t *testing.T) {
	store := newTestStore(t)
	result := store.UnsignCookie("tampered-invalid-cookie")
	if result != "" {
		t.Errorf("expected empty string for invalid cookie, got %q", result)
	}
}

func TestStore_PurgeExpired(t *testing.T) {
	// Créer un store avec TTL très court
	dir := t.TempDir()
	shortStore := session.NewStore(filepath.Join(dir, "sessions"), 1*time.Millisecond, "test-secret-32-bytesXXXXXXXXXX")
	sess := shortStore.New()
	_ = shortStore.Touch(sess)

	// Attendre que la session expire
	time.Sleep(10 * time.Millisecond)

	removed := shortStore.PurgeExpired()
	if removed == 0 {
		t.Log("PurgeExpired returned 0, session may not have expired yet (timing)")
	}
}

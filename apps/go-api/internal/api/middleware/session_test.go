// Package middleware_test — session_test.go : tests unitaires du middleware WithSession.
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/platform/session"
)

func newSessionStore(t *testing.T) *session.Store {
	t.Helper()
	dir := t.TempDir()
	return session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
}

func TestWithSession_CreatesSessionWhenNoCookie(t *testing.T) {
	store := newSessionStore(t)
	mw := middleware.WithSession(store, false)

	var capturedID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := middleware.GetSession(r.Context())
		if sess == nil {
			t.Fatal("session should be injected in context")
		}
		capturedID = sess.SessionID
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if capturedID == "" {
		t.Fatal("session ID should not be empty")
	}
	// Cookie doit être posé
	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == session.CookieName {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("session cookie should be set")
	}
}

func TestWithSession_LoadsExistingSession(t *testing.T) {
	store := newSessionStore(t)
	// Créer une session
	sess := store.New()
	_ = store.Touch(sess)
	signed := store.SignCookie(sess.SessionID)

	mw := middleware.WithSession(store, false)
	var loadedID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := middleware.GetSession(r.Context())
		if s != nil {
			loadedID = s.SessionID
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: signed})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if loadedID != sess.SessionID {
		t.Fatalf("expected session %q, got %q", sess.SessionID, loadedID)
	}
}

func TestGetSession_NilWhenAbsent(t *testing.T) {
	// Contexte vide → GetSession doit retourner nil
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess := middleware.GetSession(req.Context())
	if sess != nil {
		t.Fatal("expected nil session from empty context")
	}
}

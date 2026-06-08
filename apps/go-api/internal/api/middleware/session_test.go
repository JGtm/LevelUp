// Package middleware_test — session_test.go : tests unitaires du middleware WithSession.
package middleware_test

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/session"
)

// sessionCookie extrait le cookie de session de la réponse, ou nil.
func sessionCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == session.CookieName {
			return c
		}
	}
	return nil
}

// runWithSession exécute le middleware WithSession avec la policy donnée sur une
// requête (optionnellement TLS / X-Forwarded-Proto) et retourne la réponse.
func runWithSession(t *testing.T, policy middleware.SecureCookiePolicy, tlsOn bool, fwdProto string) *httptest.ResponseRecorder {
	t.Helper()
	store := newSessionStore(t)
	mw := middleware.WithSession(store, policy)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if tlsOn {
		req.TLS = &tls.ConnectionState{}
	}
	if fwdProto != "" {
		req.Header.Set("X-Forwarded-Proto", fwdProto)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// TestWithSession_CookieNotSecureOverHTTP est le garde de NON-RÉGRESSION du bug
// onboarding 2026-06-08 : sur HTTP nu, le cookie de session NE DOIT PAS porter le
// flag Secure (sinon le navigateur le jette sur http://localhost → session perdue
// → device flow bloqué sur attempt_not_found).
func TestWithSession_CookieNotSecureOverHTTP(t *testing.T) {
	c := sessionCookie(runWithSession(t, middleware.SecureCookiePolicy{Mode: middleware.CookieSecureAuto}, false, ""))
	if c == nil {
		t.Fatal("cookie de session attendu")
	}
	if c.Secure {
		t.Error("cookie Secure sur HTTP nu : régression du bug onboarding (navigateur jetterait le cookie)")
	}
}

func TestWithSession_CookieSecureOverTLS(t *testing.T) {
	c := sessionCookie(runWithSession(t, middleware.SecureCookiePolicy{Mode: middleware.CookieSecureAuto}, true, ""))
	if c == nil || !c.Secure {
		t.Errorf("cookie Secure attendu sur TLS, got %+v", c)
	}
}

func TestWithSession_CookieSecureViaTrustedProxy(t *testing.T) {
	policy := middleware.SecureCookiePolicy{Mode: middleware.CookieSecureAuto, TrustProxy: true}
	c := sessionCookie(runWithSession(t, policy, false, "https"))
	if c == nil || !c.Secure {
		t.Errorf("cookie Secure attendu via X-Forwarded-Proto+trust, got %+v", c)
	}
}

func TestWithSession_ForceFalseOverridesTLS(t *testing.T) {
	c := sessionCookie(runWithSession(t, middleware.SecureCookiePolicy{Mode: middleware.CookieSecureFalse}, true, ""))
	if c == nil {
		t.Fatal("cookie de session attendu")
	}
	if c.Secure {
		t.Error("Mode=false doit forcer non-Secure même sur TLS")
	}
}

func newSessionStore(t *testing.T) *session.Store {
	t.Helper()
	dir := t.TempDir()
	return session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
}

func TestWithSession_CreatesSessionWhenNoCookie(t *testing.T) {
	store := newSessionStore(t)
	mw := middleware.WithSession(store, middleware.SecureCookiePolicy{})

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

	mw := middleware.WithSession(store, middleware.SecureCookiePolicy{})
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

func TestWithSession_InjectsHaloAuth(t *testing.T) {
	store := newSessionStore(t)

	sess := store.New()
	sess.HaloTokens = &domain.HaloTokens{SpartanToken: "spartan_test", ClearanceToken: "clear_test"}
	sess.LinkedHaloIdentity = &domain.HaloIdentity{Gamertag: "TestPlayer", XUID: "xuid-42"}
	_ = store.Save(sess)
	signed := store.SignCookie(sess.SessionID)

	mw := middleware.WithSession(store, middleware.SecureCookiePolicy{})
	var gotTokens *domain.HaloTokens
	var gotXUID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTokens = ctxkeys.HaloTokens(r.Context())
		gotXUID = ctxkeys.HaloXUID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: signed})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if gotTokens == nil {
		t.Fatal("expected HaloTokens injected in context")
	}
	if gotTokens.SpartanToken != "spartan_test" {
		t.Errorf("expected spartan_test, got %q", gotTokens.SpartanToken)
	}
	if gotXUID != "xuid-42" {
		t.Errorf("expected xuid-42, got %q", gotXUID)
	}
}

func TestWithSession_NoHaloAuthWhenTokensAbsent(t *testing.T) {
	store := newSessionStore(t)

	sess := store.New()
	_ = store.Save(sess)
	signed := store.SignCookie(sess.SessionID)

	mw := middleware.WithSession(store, middleware.SecureCookiePolicy{})
	var gotTokens *domain.HaloTokens
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTokens = ctxkeys.HaloTokens(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: signed})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if gotTokens != nil {
		t.Errorf("expected nil tokens when session has no HaloTokens, got %+v", gotTokens)
	}
}

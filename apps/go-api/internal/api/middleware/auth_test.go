package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/domain"
)

// withSession injecte une SessionData dans le contexte de la requête.
func withSession(r *http.Request, sess *domain.SessionData) *http.Request {
	ctx := context.WithValue(r.Context(), sessionKey{}, sess)
	return r.WithContext(ctx)
}

// okHandler est un handler qui retourne 200 OK.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// --- RequireAuth tests ---

func TestRequireAuth_NoneMode_PassThrough(t *testing.T) {
	mw := RequireAuth(false, "none")
	handler := mw(okHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("none mode: status = %d, want 200", rr.Code)
	}
}

func TestRequireAuth_DemoMode_PassThrough(t *testing.T) {
	mw := RequireAuth(true, "password")
	handler := mw(okHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("demo mode: status = %d, want 200", rr.Code)
	}
}

func TestRequireAuth_PasswordMode_NoSession_401(t *testing.T) {
	mw := RequireAuth(false, "password")
	handler := mw(okHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("no session: status = %d, want 401", rr.Code)
	}
}

func TestRequireAuth_PasswordMode_NoUsername_401(t *testing.T) {
	mw := RequireAuth(false, "password")
	handler := mw(okHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	sess := &domain.SessionData{SessionID: "test-sess"}
	req = withSession(req, sess)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("no username: status = %d, want 401", rr.Code)
	}
}

func TestRequireAuth_PasswordMode_WithUsername_200(t *testing.T) {
	mw := RequireAuth(false, "password")
	handler := mw(okHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	username := "alice"
	sess := &domain.SessionData{SessionID: "test-sess", Username: &username}
	req = withSession(req, sess)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("with username: status = %d, want 200", rr.Code)
	}
}

// --- RequireAdmin tests ---

func TestRequireAdmin_NoSession_403(t *testing.T) {
	mw := RequireAdmin(false, "password")
	handler := mw(okHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/test", nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("no session: status = %d, want 403", rr.Code)
	}
}

func TestRequireAdmin_UserRole_403(t *testing.T) {
	mw := RequireAdmin(false, "password")
	handler := mw(okHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/test", nil)
	role := "user"
	sess := &domain.SessionData{SessionID: "test-sess", Role: &role}
	req = withSession(req, sess)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("user role: status = %d, want 403", rr.Code)
	}
}

func TestRequireAdmin_AdminRole_200(t *testing.T) {
	mw := RequireAdmin(false, "password")
	handler := mw(okHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/test", nil)
	role := "admin"
	sess := &domain.SessionData{SessionID: "test-sess", Role: &role}
	req = withSession(req, sess)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("admin role: status = %d, want 200", rr.Code)
	}
}

func TestRequireAdmin_AuthModeNone_200(t *testing.T) {
	// En auth_mode=none (usage local sans auth), RequireAdmin doit être transparent.
	mw := RequireAdmin(false, "none")
	handler := mw(okHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/watcher/status", nil)
	// Pas de session injectée — simule un accès local sans authentification.
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("auth_mode=none sans session: status = %d, want 200", rr.Code)
	}
}

func TestRequireAdmin_DemoMode_200(t *testing.T) {
	// En mode démo, RequireAdmin doit être transparent même sans session.
	mw := RequireAdmin(true, "password")
	handler := mw(okHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/users", nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("demo mode sans session: status = %d, want 200", rr.Code)
	}
}

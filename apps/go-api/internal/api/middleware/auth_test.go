package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

// En démo, RequireAuth REFUSE (403 demo_mode_forbidden) sauf les lectures de la liste nommée
// demoAuthOnlyReads / demoAdminReads (demo_guarded_reads.go).
func TestRequireAuth_DemoMode_RefusesExceptNamedReads(t *testing.T) {
	handler := RequireAuth(true, "password")(okHandler)
	cases := []struct {
		method, path string
		want         int
	}{
		{"GET", "/api/v1/presence", http.StatusOK},
		{"GET", "/api/v1/settings", http.StatusOK},
		{"GET", "/api/v1/healthz/home", http.StatusOK},
		{"GET", "/api/v1/groups", http.StatusForbidden},
		{"GET", "/api/v1/jobs/j1", http.StatusForbidden},
		{"GET", "/api/v1/_diag/csr-coverage/demo-player", http.StatusForbidden},
		{"GET", "/api/v1/presence/", http.StatusForbidden},
		{"POST", "/api/v1/presence", http.StatusForbidden},
		{"PATCH", "/api/v1/settings", http.StatusForbidden},
		{"GET", "/api/v1/titles/halo_infinite/catalog/maps", http.StatusForbidden},
	}
	for _, c := range cases {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(c.method, c.path, nil))
		if rr.Code != c.want {
			t.Errorf("démo %s %s : status = %d, want %d", c.method, c.path, rr.Code, c.want)
		}
		if c.want == http.StatusForbidden && !strings.Contains(rr.Body.String(), DemoModeForbiddenCode) {
			t.Errorf("démo %s %s : corps %q sans le code %s", c.method, c.path, rr.Body.String(), DemoModeForbiddenCode)
		}
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
	mw := RequireAdmin(false, "password", adminAccounts{"boss": domain.RoleAdmin})
	handler := mw(okHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/test", nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("no session: status = %d, want 403", rr.Code)
	}
}

func TestRequireAdmin_UserRole_403(t *testing.T) {
	mw := RequireAdmin(false, "password", adminAccounts{"boss": domain.RoleAdmin})
	handler := mw(okHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/test", nil)
	role := "user"
	boss := "bob" // compte au rôle utilisateur (absent des admins)
	sess := &domain.SessionData{SessionID: "test-sess", Username: &boss, Role: &role}
	req = withSession(req, sess)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("user role: status = %d, want 403", rr.Code)
	}
}

func TestRequireAdmin_AdminRole_200(t *testing.T) {
	mw := RequireAdmin(false, "password", adminAccounts{"boss": domain.RoleAdmin})
	handler := mw(okHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/test", nil)
	role := "admin"
	boss := "boss"
	sess := &domain.SessionData{SessionID: "test-sess", Username: &boss, Role: &role}
	req = withSession(req, sess)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("admin role: status = %d, want 200", rr.Code)
	}
}

func TestRequireAdmin_AuthModeNone_200(t *testing.T) {
	// En auth_mode=none (usage local sans auth), RequireAdmin doit être transparent.
	mw := RequireAdmin(false, "none", nil)
	handler := mw(okHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/watcher/status", nil)
	// Pas de session injectée — simule un accès local sans authentification.
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("auth_mode=none sans session: status = %d, want 200", rr.Code)
	}
}

// En démo, RequireAdmin REFUSE (403 demo_mode_forbidden) sauf les deux lectures de demoAdminReads
// (réglages lus par l'interface, sonde de la home des e2e et du harnais) ; une session admin ne
// change rien : la démo n'a pas de compte, et un visiteur n'est jamais administrateur.
func TestRequireAdmin_DemoMode_RefusesExceptNamedReads(t *testing.T) {
	handler := RequireAdmin(true, "password", nil)(okHandler)
	cases := []struct {
		method, path string
		want         int
	}{
		{"GET", "/api/v1/settings", http.StatusOK},
		{"GET", "/api/v1/healthz/home", http.StatusOK},
		{"GET", "/api/v1/presence", http.StatusForbidden},
		{"GET", "/api/v1/admin/users", http.StatusForbidden},
		{"GET", "/api/v1/admin/monitoring/logs/tail", http.StatusForbidden},
		{"GET", "/api/v1/watcher/status", http.StatusForbidden},
		{"GET", "/debug/vars", http.StatusForbidden},
		{"GET", "/debug/vars/", http.StatusForbidden},
		{"PATCH", "/api/v1/settings", http.StatusForbidden},
		{"POST", "/api/v1/healthz/home", http.StatusForbidden},
		{"GET", "/api/v1/settings/backup/status", http.StatusForbidden},
		{"GET", "/api/v1/../admin/users", http.StatusForbidden},
	}
	for _, c := range cases {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(c.method, c.path, nil))
		if rr.Code != c.want {
			t.Errorf("démo %s %s : status = %d, want %d", c.method, c.path, rr.Code, c.want)
		}
		if c.want == http.StatusForbidden && !strings.Contains(rr.Body.String(), DemoModeForbiddenCode) {
			t.Errorf("démo %s %s : corps %q sans le code %s", c.method, c.path, rr.Body.String(), DemoModeForbiddenCode)
		}
	}
}

// Le chemin que la garde lit doit être celui que chi route : un chemin ambigu (encodé ou
// dénormalisé) n'entre jamais dans la liste nommée.
func TestRequireAdmin_DemoMode_CheminAmbiguNeContourneJamais(t *testing.T) {
	handler := RequireAdmin(true, "password", nil)(okHandler)
	for _, target := range []string{
		"/api/v1/settings%2Fx", "/api/v1/%2e%2e/settings", "/api/v1//settings", "/api/v1/./settings",
		"/api/v1/%73ettings", "/api/v1/settings/",
	} {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest("GET", target, nil))
		if rr.Code != http.StatusForbidden {
			t.Errorf("démo GET %s : status = %d, want 403", target, rr.Code)
		}
	}
}

// adminAccounts : comptes des tests de RequireAdmin, rôle par nom d'utilisateur.
type adminAccounts map[string]domain.UserRole

func (a adminAccounts) Get(username string) (*domain.User, error) {
	if role, ok := a[username]; ok {
		return &domain.User{Username: username, Role: role}, nil
	}
	return nil, errors.New("utilisateur introuvable")
}

func (a adminAccounts) GetByXUID(string) (*domain.User, error) {
	return nil, errors.New("utilisateur introuvable")
}

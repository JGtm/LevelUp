// Package handlers_test — admin_reset_password_revokes_test.go : la réinitialisation du mot
// de passe d'un compte par l'admin ferme les sessions de CE compte (doctrine D5, lot L9-7).
package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/platform/userstore"
)

const (
	resetAdminUser = "chef"
	resetAdminPass = "Admin-Mot-De-Passe-9"
	resetTargetNew = "Reinitialise-Mot-De-Passe-4"
)

// resetRig : un admin, deux comptes ordinaires, l'API de connexion, la route d'admin
// (RequireAuth + RequireAdmin) et une route de lecture gardée par RequireAuth.
func resetRig(t *testing.T) (http.Handler, *session.Store) {
	t.Helper()
	dir := t.TempDir()
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	for _, u := range []struct {
		name, pass string
		role       domain.UserRole
	}{
		{resetAdminUser, resetAdminPass, domain.RoleAdmin},
		{loginLimitUser, loginLimitPassword, domain.RoleUser},
		{"voisin", "Voisin-Mot-De-Passe-3", domain.RoleUser},
	} {
		if _, err := users.Create(u.name, u.pass, u.role); err != nil {
			t.Fatalf("création du compte %s : %v", u.name, err)
		}
	}
	invites := userstore.NewInviteStore(filepath.Join(dir, "invites.json"), users)
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")

	auth := handlers.NewUserAuthHandler(users, invites, sessStore, "open").WithAuthMode("password")
	admin := handlers.NewAdminHandler(users, invites).WithSessions(sessStore)

	r := chi.NewRouter()
	r.Use(chimiddleware.ClientIPFromRemoteAddr)
	r.Use(chimiddleware.ClientIPFromHeader("X-Real-IP"))
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	auth.Mount(r)
	r.With(middleware.RequireAuth(false, "password")).Get("/me", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Route("/admin", func(r chi.Router) {
		r.Use(middleware.RequireAuth(false, "password"))
		r.Use(middleware.RequireAdmin(false, "password", users))
		admin.Mount(r)
	})
	return r, sessStore
}

func resetPasswordAs(r http.Handler, cookie *http.Cookie, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPatch, "/admin/users/"+target+"/password",
		strings.NewReader(`{"new_password":"`+resetTargetNew+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// Réinitialiser le mot de passe d'un compte : ses sessions ouvertes (plusieurs appareils)
// répondent 401 et disparaissent du serveur ; les sessions des autres comptes et celle de
// l'admin restent valables.
func TestAdminResetPassword_RevokesTargetSessions(t *testing.T) {
	r, sessStore := resetRig(t)
	adminC := sessionCookie(t, loginFrom(t, r, "192.0.2.10", resetAdminUser, resetAdminPass))
	target1 := sessionCookie(t, loginFrom(t, r, "192.0.2.11", loginLimitUser, loginLimitPassword))
	target2 := sessionCookie(t, loginFrom(t, r, "192.0.2.12", loginLimitUser, loginLimitPassword))
	neighbour := sessionCookie(t, loginFrom(t, r, "192.0.2.13", "voisin", "Voisin-Mot-De-Passe-3"))
	target1ID := sessStore.UnsignCookie(target1.Value)
	for name, c := range map[string]*http.Cookie{"cible 1": target1, "cible 2": target2, "voisin": neighbour, "admin": adminC} {
		if got := serveWithCookie(r, http.MethodGet, "/me", c).Code; got != http.StatusOK {
			t.Fatalf("session %s avant la réinitialisation : %d, attendu 200", name, got)
		}
	}

	if w := resetPasswordAs(r, adminC, loginLimitUser); w.Code != http.StatusNoContent {
		t.Fatalf("réinitialisation : %d %s, attendu 204", w.Code, w.Body.String())
	}

	for name, c := range map[string]*http.Cookie{"cible 1": target1, "cible 2": target2} {
		if got := serveWithCookie(r, http.MethodGet, "/me", c).Code; got != http.StatusUnauthorized {
			t.Errorf("session %s après la réinitialisation : %d, attendu 401", name, got)
		}
	}
	if sessStore.Load(t.Context(), target1ID) != nil {
		t.Error("la session du compte réinitialisé existe encore côté serveur")
	}
	for name, c := range map[string]*http.Cookie{"voisin": neighbour, "admin": adminC} {
		if got := serveWithCookie(r, http.MethodGet, "/me", c).Code; got != http.StatusOK {
			t.Errorf("session %s après la réinitialisation : %d, attendu 200", name, got)
		}
	}
	// Le nouveau mot de passe ouvre une session neuve.
	if got := loginFrom(t, r, "192.0.2.14", loginLimitUser, resetTargetNew).Code; got != http.StatusOK {
		t.Errorf("connexion avec le mot de passe réinitialisé : %d, attendu 200", got)
	}
}

// L'admin qui réinitialise SON PROPRE mot de passe par cette route garde sa session.
func TestAdminResetPassword_OwnAccountKeepsCurrentSession(t *testing.T) {
	r, _ := resetRig(t)
	adminC := sessionCookie(t, loginFrom(t, r, "192.0.2.20", resetAdminUser, resetAdminPass))
	other := sessionCookie(t, loginFrom(t, r, "192.0.2.21", resetAdminUser, resetAdminPass))

	if w := resetPasswordAs(r, adminC, resetAdminUser); w.Code != http.StatusNoContent {
		t.Fatalf("réinitialisation de son propre mot de passe : %d %s, attendu 204", w.Code, w.Body.String())
	}
	if got := serveWithCookie(r, http.MethodGet, "/me", adminC).Code; got != http.StatusOK {
		t.Errorf("session courante de l'admin : %d, attendu 200", got)
	}
	if got := serveWithCookie(r, http.MethodGet, "/me", other).Code; got != http.StatusUnauthorized {
		t.Errorf("autre session de l'admin : %d, attendu 401", got)
	}
}

// Package handlers_test — user_auth_password_revokes_test.go : changer son mot de passe
// ferme les AUTRES sessions du compte. La session qui a fait le changement est conservée,
// sous un nouvel identifiant.
package handlers_test

import (
	"net/http"
	"path/filepath"
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

// passwordRevokeRig : un compte avec mot de passe, une route de lecture gardée par
// RequireAuth, et le magasin de sessions pour inspecter l'état serveur.
func passwordRevokeRig(t *testing.T) (http.Handler, *session.Store, *userstore.Store) {
	t.Helper()
	dir := t.TempDir()
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	if _, err := users.Create(loginLimitUser, loginLimitPassword, domain.RoleUser); err != nil {
		t.Fatalf("création du compte de test : %v", err)
	}
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	h := handlers.NewUserAuthHandler(users, userstore.NewInviteStore(filepath.Join(dir, "invites.json"), users), sessStore, "open").
		WithAuthMode("password")
	r := chi.NewRouter()
	r.Use(chimiddleware.ClientIPFromRemoteAddr)
	r.Use(chimiddleware.ClientIPFromHeader("X-Real-IP"))
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	h.Mount(r)
	r.With(middleware.RequireAuth(false, "password")).Get("/me", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return r, sessStore, users
}

// sessionCookie extrait le cookie de session posé par une réponse.
func sessionCookie(t *testing.T, w interface{ Result() *http.Response }) *http.Cookie {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == session.CookieName && c.MaxAge >= 0 {
			return c
		}
	}
	t.Fatal("aucun cookie de session dans la réponse")
	return nil
}

// TestSetPassword_RevokesOtherSessions : deux sessions du même compte ; le changement de
// mot de passe depuis l'une ferme l'autre (lecture refusée, fichier supprimé) et garde
// la première, sous un nouvel identifiant.
func TestSetPassword_RevokesOtherSessions(t *testing.T) {
	r, sessStore, _ := passwordRevokeRig(t)
	const ip = "192.0.2.90"
	current := sessionCookie(t, loginFrom(t, r, ip, loginLimitUser, loginLimitPassword))
	other := sessionCookie(t, loginFrom(t, r, "192.0.2.91", loginLimitUser, loginLimitPassword))
	otherID := sessStore.UnsignCookie(other.Value)
	for name, c := range map[string]*http.Cookie{"courante": current, "autre": other} {
		if got := serveWithCookie(r, http.MethodGet, "/me", c).Code; got != http.StatusOK {
			t.Fatalf("session %s avant le changement : %d, attendu 200", name, got)
		}
	}

	w := postFromIP(t, r, "/auth/password", ip,
		setPasswordBody{Password: nouveauMotDePasse, CurrentPassword: loginLimitPassword}, current)
	if w.Code != http.StatusNoContent {
		t.Fatalf("changement de mot de passe : %d %s, attendu 204", w.Code, w.Body.String())
	}
	rotated := sessionCookie(t, w)
	if rotated.Value == current.Value {
		t.Error("la session courante garde son identifiant après le changement de mot de passe")
	}
	if got := serveWithCookie(r, http.MethodGet, "/me", rotated).Code; got != http.StatusOK {
		t.Errorf("session courante (nouvel identifiant) : %d, attendu 200", got)
	}
	if got := serveWithCookie(r, http.MethodGet, "/me", other).Code; got != http.StatusUnauthorized {
		t.Errorf("autre session après le changement : %d, attendu 401", got)
	}
	if sess := sessStore.Load(t.Context(), otherID); sess != nil {
		t.Error("l'autre session existe encore côté serveur")
	}
}

// TestSetPassword_LeavesOtherAccountsSessions : les sessions d'un autre compte ne sont
// pas touchées.
func TestSetPassword_LeavesOtherAccountsSessions(t *testing.T) {
	r, _, users := passwordRevokeRig(t)
	const otherUser, otherPass = "voisin", "Voisin-Mot-De-Passe-3"
	if _, err := users.Create(otherUser, otherPass, domain.RoleUser); err != nil {
		t.Fatalf("création du second compte : %v", err)
	}
	neighbour := sessionCookie(t, loginFrom(t, r, "192.0.2.92", otherUser, otherPass))
	current := sessionCookie(t, loginFrom(t, r, "192.0.2.93", loginLimitUser, loginLimitPassword))

	w := postFromIP(t, r, "/auth/password", "192.0.2.93",
		setPasswordBody{Password: nouveauMotDePasse, CurrentPassword: loginLimitPassword}, current)
	if w.Code != http.StatusNoContent {
		t.Fatalf("changement de mot de passe : %d %s, attendu 204", w.Code, w.Body.String())
	}
	if got := serveWithCookie(r, http.MethodGet, "/me", neighbour).Code; got != http.StatusOK {
		t.Errorf("session d'un autre compte : %d, attendu 200", got)
	}
}

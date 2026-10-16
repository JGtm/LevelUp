// Package handlers_test — user_auth_logout_test.go : la déconnexion supprime la
// session côté serveur et expire le cookie ; le même cookie n'ouvre plus rien.
package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/platform/userstore"
)

// ssoSessionRig : une session telle que la laisse une connexion SSO Xbox (compte,
// rôle, AuthReady, identité Halo), et une route de lecture gardée par RequireAuth.
func ssoSessionRig(t *testing.T) (*chi.Mux, *session.Store, *http.Cookie, string) {
	t.Helper()
	dir := t.TempDir()
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	h := handlers.NewUserAuthHandler(users, userstore.NewInviteStore(filepath.Join(dir, "invites.json"), users), sessStore, "invite").
		WithAuthMode("xbox")

	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	h.Mount(r)
	r.With(middleware.RequireAuth(false, "xbox")).Get("/players/me/matches", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	sess := sessStore.New()
	username, role := "jgtm", string(domain.RoleAdmin)
	sess.Username, sess.Role = &username, &role
	sess.AuthReady = true
	sess.LinkedHaloIdentity = &domain.HaloIdentity{Gamertag: "JGtm", XUID: "2535400000000042"}
	if err := sessStore.Save(sess); err != nil {
		t.Fatalf("save session : %v", err)
	}
	return r, sessStore, &http.Cookie{Name: session.CookieName, Value: sessStore.SignCookie(sess.SessionID)}, sess.SessionID
}

func serveWithCookie(r http.Handler, method, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestLogout_SSOSessionNoLongerReads : SSO → déconnexion → lecture refusée, session
// supprimée côté serveur, cookie expiré dans la réponse.
func TestLogout_SSOSessionNoLongerReads(t *testing.T) {
	r, sessStore, cookie, sessionID := ssoSessionRig(t)

	if got := serveWithCookie(r, http.MethodGet, "/players/me/matches", cookie).Code; got != http.StatusOK {
		t.Fatalf("lecture avant déconnexion : %d, attendu 200", got)
	}

	w := serveWithCookie(r, http.MethodPost, "/auth/logout", cookie)
	if w.Code != http.StatusNoContent {
		t.Fatalf("logout : %d, attendu 204", w.Code)
	}
	expired := false
	for _, c := range w.Result().Cookies() {
		if c.Name == session.CookieName {
			expired = c.MaxAge < 0
		}
	}
	if !expired {
		t.Error("la réponse de déconnexion n'expire pas le cookie de session")
	}
	if sess := sessStore.Load(t.Context(), sessionID); sess != nil {
		t.Error("la session existe encore côté serveur après la déconnexion")
	}

	if got := serveWithCookie(r, http.MethodGet, "/players/me/matches", cookie).Code; got != http.StatusUnauthorized {
		t.Errorf("lecture avec le cookie d'avant la déconnexion : %d, attendu 401", got)
	}
}

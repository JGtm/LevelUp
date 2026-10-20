// Package handlers_test — session_rotation_test.go : l'identifiant de session est
// renouvelé à chaque connexion (registre sécurité C5-4). Un cookie planté avant la
// connexion ne vaut plus rien après ; l'état de navigation suit la nouvelle session.
package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/service"
)

// plantedSession : une session anonyme déjà persistée (cookie planté), qui porte un
// état de navigation (titre courant).
func plantedSession(t *testing.T, sessStore *session.Store) (*domain.SessionData, *http.Cookie) {
	t.Helper()
	sess := sessStore.New()
	sess.CurrentTitleSlug = "halo_5"
	if err := sessStore.Save(sess); err != nil {
		t.Fatalf("save : %v", err)
	}
	return sess, &http.Cookie{Name: session.CookieName, Value: sessStore.SignCookie(sess.SessionID)}
}

// responseSessionID rend l'identifiant porté par le cookie de session de la réponse.
func responseSessionID(t *testing.T, sessStore *session.Store, w *httptest.ResponseRecorder) string {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == session.CookieName && c.MaxAge >= 0 {
			return sessStore.UnsignCookie(c.Value)
		}
	}
	return ""
}

// assertRotated : nouveau cookie, ancienne session supprimée, nouvelle session
// authentifiée reprenant l'état de navigation.
func assertRotated(t *testing.T, sessStore *session.Store, w *httptest.ResponseRecorder, planted *domain.SessionData) *domain.SessionData {
	t.Helper()
	newID := responseSessionID(t, sessStore, w)
	if newID == "" || newID == planted.SessionID {
		t.Fatalf("identifiant de session après connexion = %q, attendu un nouvel identifiant (avant : %q)", newID, planted.SessionID)
	}
	if sessStore.Load(t.Context(), planted.SessionID) != nil {
		t.Error("l'ancienne session existe encore après la connexion")
	}
	fresh := sessStore.Load(t.Context(), newID)
	if fresh == nil {
		t.Fatal("nouvelle session introuvable")
	}
	if fresh.CurrentTitleSlug != "halo_5" {
		t.Errorf("état de navigation perdu à la rotation : titre = %q", fresh.CurrentTitleSlug)
	}
	return fresh
}

func TestRotation_PasswordLogin(t *testing.T) {
	dir := t.TempDir()
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	if _, err := users.Create("jgtm", testUserAuthPass, domain.RoleAdmin); err != nil {
		t.Fatalf("Create : %v", err)
	}
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	h := handlers.NewUserAuthHandler(users, userstore.NewInviteStore(filepath.Join(dir, "invites.json"), users), sessStore, "invite").
		WithAuthMode("xbox")
	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	h.Mount(r)

	planted, cookie := plantedSession(t, sessStore)
	body, _ := json.Marshal(domain.LoginRequest{Username: "jgtm", Password: testUserAuthPass})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login : %d (%s)", w.Code, w.Body.String())
	}
	if fresh := assertRotated(t, sessStore, w, planted); fresh.Username == nil || *fresh.Username != "jgtm" {
		t.Errorf("nouvelle session non authentifiée : %+v", fresh.Username)
	}
}

func TestRotation_DeviceFlow(t *testing.T) {
	dir := t.TempDir()
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	attempts := auth_platform.NewAttemptStore()
	h := handlers.NewAuthHandler(sessStore, attempts, &stubTokenProvider{}).
		WithLinkStrategy(service.NewXboxSSOLinkStrategy(users))
	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	h.MountDeviceFlow(r)

	planted, cookie := plantedSession(t, sessStore)
	attempt, _, _ := attempts.GetOrCreate(planted.SessionID, "")
	attempts.Update(attempt.AttemptID, func(a *auth_platform.Attempt) {
		a.Status = auth_platform.AttemptStatusAuthorized
		a.Gamertag, a.XUID = "Rotator", "2535400000000077"
	})

	poll := func(c *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/auth/device-flow/"+attempt.AttemptID, nil)
		req.AddCookie(c)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	w := poll(cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("statut : %d (%s)", w.Code, w.Body.String())
	}
	fresh := assertRotated(t, sessStore, w, planted)
	if !fresh.AuthReady || fresh.Username == nil {
		t.Errorf("nouvelle session non authentifiée : AuthReady=%v", fresh.AuthReady)
	}

	// Le sondage suivant, avec le NOUVEAU cookie, retrouve la tentative (le front ne
	// la perd pas) et ne reconnecte pas une seconde fois.
	next := poll(&http.Cookie{Name: session.CookieName, Value: sessStore.SignCookie(fresh.SessionID)})
	if next.Code != http.StatusOK {
		t.Fatalf("sondage avec le nouveau cookie : %d (%s)", next.Code, next.Body.String())
	}
	if again := responseSessionID(t, sessStore, next); again != fresh.SessionID {
		t.Errorf("second sondage : identifiant %q, attendu inchangé %q", again, fresh.SessionID)
	}
}

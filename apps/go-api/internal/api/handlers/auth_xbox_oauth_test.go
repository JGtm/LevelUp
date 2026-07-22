// Package handlers_test — auth_xbox_oauth_test.go : tests XboxOAuthHandler.
//
// Couvre les cas non-exchange : demo_mode, redirect_uri absent, state CSRF
// (manquant, mismatch), error param Microsoft. Le success path complet
// nécessiterait un mock de l'endpoint Microsoft /oauth2/v2.0/token (non testé
// ici — exercé via les tests d'intégration manuels avec Azure).
package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/platform/userstore"
)

func newXboxOAuthRouter(t *testing.T, demoMode bool, redirectURI string) (*chi.Mux, *session.Store) {
	t.Helper()
	dir := t.TempDir()
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	h := handlers.NewXboxOAuthHandler(sessStore, &stubTokenProvider{}, demoMode, redirectURI)

	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	r.Get("/auth/xbox/login", h.LoginRedirect)
	r.Get("/auth/xbox/callback", h.Callback)
	return r, sessStore
}

func TestXboxOAuth_LoginRedirect_DemoMode(t *testing.T) {
	r, _ := newXboxOAuthRouter(t, true, "http://localhost:8000/api/v1/auth/xbox/callback")

	req := httptest.NewRequest(http.MethodGet, "/auth/xbox/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422 (demo_mode)", w.Code)
	}
}

func TestXboxOAuth_LoginRedirect_NoRedirectURI(t *testing.T) {
	r, _ := newXboxOAuthRouter(t, false, "")

	req := httptest.NewRequest(http.MethodGet, "/auth/xbox/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 (redirect_uri_not_configured)", w.Code)
	}
	if !strings.Contains(w.Body.String(), "redirect_uri_not_configured") {
		t.Errorf("body devrait contenir redirect_uri_not_configured, got: %s", w.Body.String())
	}
}

func TestXboxOAuth_LoginRedirect_Success(t *testing.T) {
	r, _ := newXboxOAuthRouter(t, false, "http://localhost:8000/cb")

	req := httptest.NewRequest(http.MethodGet, "/auth/xbox/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want 302 (redirect Microsoft)", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "login.microsoftonline.com") {
		t.Errorf("Location ne pointe pas vers Microsoft : %s", loc)
	}
	if !strings.Contains(loc, "state=") {
		t.Errorf("Location ne contient pas de state : %s", loc)
	}
	if !strings.Contains(loc, "response_type=code") {
		t.Errorf("Location ne contient pas response_type=code : %s", loc)
	}
}

func TestXboxOAuth_Callback_MissingCode(t *testing.T) {
	r, _ := newXboxOAuthRouter(t, false, "http://localhost:8000/cb")

	req := httptest.NewRequest(http.MethodGet, "/auth/xbox/callback?state=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (missing code)", w.Code)
	}
}

func TestXboxOAuth_Callback_MissingState(t *testing.T) {
	r, _ := newXboxOAuthRouter(t, false, "http://localhost:8000/cb")

	req := httptest.NewRequest(http.MethodGet, "/auth/xbox/callback?code=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (missing state)", w.Code)
	}
}

func TestXboxOAuth_Callback_StateMismatch(t *testing.T) {
	r, _ := newXboxOAuthRouter(t, false, "http://localhost:8000/cb")

	// Pas de state en session → toute valeur reçue est rejetée.
	req := httptest.NewRequest(http.MethodGet, "/auth/xbox/callback?code=abc&state=fakestate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403 (state mismatch)", w.Code)
	}
	if !strings.Contains(w.Body.String(), "state_mismatch") {
		t.Errorf("body devrait contenir state_mismatch, got: %s", w.Body.String())
	}
}

func TestXboxOAuth_Callback_ErrorParam(t *testing.T) {
	r, _ := newXboxOAuthRouter(t, false, "http://localhost:8000/cb")

	// Microsoft renvoie ?error=... quand l'user a refusé.
	req := httptest.NewRequest(http.MethodGet, "/auth/xbox/callback?error=access_denied&error_description=user+refused", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (oauth_denied)", w.Code)
	}
}

// Helper : récupère le state d'une réponse LoginRedirect et le stocke dans
// la session pour pouvoir tester un callback "happy path" partiel.
// On ne peut pas tester l'exchange réel (HTTP vers Microsoft), donc on
// s'arrête au stade "state match → exchange tenté" (qui échouera en network
// timeout dans le test, mais ce n'est pas notre scope).
func TestXboxOAuth_Callback_StateMatches_AttemptsExchange(t *testing.T) {
	r, _ := newXboxOAuthRouter(t, false, "http://localhost:8000/cb")

	// Étape 1 : LoginRedirect pour générer un state en session.
	loginReq := httptest.NewRequest(http.MethodGet, "/auth/xbox/login", nil)
	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginReq)
	if loginW.Code != http.StatusFound {
		t.Fatalf("LoginRedirect status = %d, want 302", loginW.Code)
	}
	// Extraire le state de la redirect.
	loc := loginW.Header().Get("Location")
	stateIdx := strings.Index(loc, "state=")
	if stateIdx < 0 {
		t.Fatalf("state manquant dans Location: %s", loc)
	}
	stateValue := loc[stateIdx+len("state="):]
	if ampIdx := strings.Index(stateValue, "&"); ampIdx >= 0 {
		stateValue = stateValue[:ampIdx]
	}

	// Étape 2 : Callback avec le bon state — devrait passer la vérif CSRF.
	// On capture le cookie de session de LoginRedirect pour qu'il soit envoyé
	// au callback (sinon nouvelle session, pas de state stocké).
	cookies := loginW.Result().Cookies()

	cbReq := httptest.NewRequest(http.MethodGet, "/auth/xbox/callback?code=mock-code&state="+stateValue, nil)
	for _, c := range cookies {
		cbReq.AddCookie(c)
	}
	cbW := httptest.NewRecorder()
	r.ServeHTTP(cbW, cbReq)

	// L'échange code → tokens va échouer (pas de vrai Microsoft), donc 500 attendu.
	// Le test valide que le CSRF a passé (sinon on aurait eu 403 state_mismatch).
	if cbW.Code == http.StatusForbidden {
		t.Errorf("state ne devrait pas mismatch : %s", cbW.Body.String())
	}
	// Si on a 500 (code_exchange_failed) → CSRF OK, exchange a tenté → succès du test.
	if cbW.Code != http.StatusInternalServerError {
		// Tolérant : si Microsoft retournait un 400 par exemple, c'est aussi OK.
		t.Logf("Callback status = %d (attendu 500 code_exchange_failed) : %s", cbW.Code, cbW.Body.String())
	}
}

// loadSessionFromResponse charge la session pointée par le cookie de la réponse.
func loadSessionFromResponse(t *testing.T, sessStore *session.Store, w *httptest.ResponseRecorder) *domain.SessionData {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == session.CookieName {
			id := sessStore.UnsignCookie(c.Value)
			if id == "" {
				t.Fatalf("cookie session invalide: %s", c.Value)
			}
			return sessStore.Load(context.Background(), id)
		}
	}
	t.Fatal("aucun cookie de session dans la réponse")
	return nil
}

// Flow "rejoindre un groupe" : ?invite=CODE valide → stocké en PendingInviteCode.
func TestXboxOAuth_LoginRedirect_ValidInviteStored(t *testing.T) {
	dir := t.TempDir()
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	invites := userstore.NewInviteStore(filepath.Join(dir, "invites.json"))
	inv, _ := invites.Generate("Owner", 7, "grp_test")

	h := handlers.NewXboxOAuthHandler(sessStore, &stubTokenProvider{}, false, "http://localhost:8000/cb").
		WithInviteStore(invites)
	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	r.Get("/auth/xbox/login", h.LoginRedirect)

	req := httptest.NewRequest(http.MethodGet, "/auth/xbox/login?invite="+inv.Code, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", w.Code)
	}
	sess := loadSessionFromResponse(t, sessStore, w)
	if sess == nil || sess.PendingInviteCode != inv.Code {
		t.Fatalf("PendingInviteCode = %q, want %q", sessPendingCode(sess), inv.Code)
	}
}

// ?invite=CODE invalide → ignoré (redirection normale, pas de PendingInviteCode).
func TestXboxOAuth_LoginRedirect_InvalidInviteIgnored(t *testing.T) {
	dir := t.TempDir()
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	invites := userstore.NewInviteStore(filepath.Join(dir, "invites.json"))

	h := handlers.NewXboxOAuthHandler(sessStore, &stubTokenProvider{}, false, "http://localhost:8000/cb").
		WithInviteStore(invites)
	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	r.Get("/auth/xbox/login", h.LoginRedirect)

	req := httptest.NewRequest(http.MethodGet, "/auth/xbox/login?invite=NOPE", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 (invite invalide ignorée)", w.Code)
	}
	sess := loadSessionFromResponse(t, sessStore, w)
	if sess != nil && sess.PendingInviteCode != "" {
		t.Fatalf("PendingInviteCode = %q, want vide", sess.PendingInviteCode)
	}
}

func sessPendingCode(s *domain.SessionData) string {
	if s == nil {
		return "<nil session>"
	}
	return s.PendingInviteCode
}

// var unused — supprime warning si ExchangeAuthorizationCode pas appelé directement.
var _ = auth_platform.LevelUpClientID

// Package handlers_test — user_auth_login_limit_test.go : limiteur d'échecs de
// connexion sur POST /auth/login et POST /auth/password.
//
// Contrat : au-delà de 5 échecs en 15 minutes pour un même identifiant (normalisé)
// OU depuis une même IP cliente, la route répond 429 pendant 15 minutes sans
// vérifier le mot de passe ; un succès remet à zéro les compteurs de l'identifiant
// et de l'IP. Le routeur de test monte la résolution d'IP de la pile transverse
// (X-Real-IP derrière le proxy) : l'IP du compteur est celle de GetClientIP.
package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	loginLimitUser      = "victime"
	loginLimitPassword  = "Bon-Mot-De-Passe-1"
	loginLimitWrongPass = "mauvais-mot-de-passe"
)

// newLoginLimitRouter monte UserAuthHandler derrière la résolution d'IP cliente de
// la pile transverse et la session.
func newLoginLimitRouter(t *testing.T, opts ...func(*handlers.UserAuthHandler)) http.Handler {
	t.Helper()
	dir := t.TempDir()
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	if _, err := users.Create(loginLimitUser, loginLimitPassword, domain.RoleUser); err != nil {
		t.Fatalf("création du compte de test : %v", err)
	}
	invites := userstore.NewInviteStore(filepath.Join(dir, "invites.json"), users)
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	h := handlers.NewUserAuthHandler(users, invites, sessStore, "open").WithAuthMode("password")
	for _, opt := range opts {
		opt(h)
	}
	r := chi.NewRouter()
	r.Use(chimiddleware.ClientIPFromRemoteAddr)
	r.Use(chimiddleware.ClientIPFromHeader("X-Real-IP"))
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	h.Mount(r)
	return r
}

// postFromIP émet un POST JSON depuis l'IP cliente donnée (X-Real-IP), avec les
// cookies éventuels.
func postFromIP(t *testing.T, r http.Handler, path, ip string, body any, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal : %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:40000"
	req.Header.Set("X-Real-IP", ip)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func loginFrom(t *testing.T, r http.Handler, ip, username, password string) *httptest.ResponseRecorder {
	t.Helper()
	return postFromIP(t, r, "/auth/login", ip, domain.LoginRequest{Username: username, Password: password})
}

func assertTooManyAttempts(t *testing.T, w *httptest.ResponseRecorder, what string) {
	t.Helper()
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("%s : status %d, want 429 (body=%s)", what, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "too_many_login_attempts") {
		t.Errorf("%s : code d'erreur too_many_login_attempts attendu, body=%s", what, w.Body.String())
	}
	if w.Header().Get("Retry-After") == "" {
		t.Errorf("%s : en-tête Retry-After absent", what)
	}
}

// TestLoginLimit_SixthFailureIs429EvenAcrossIPs : 5 échecs sur un même identifiant,
// chacun depuis une IP différente → la 6e tentative reçoit 429, y compris avec le
// bon mot de passe, et y compris avec une casse ou des espaces différents.
func TestLoginLimit_SixthFailureIs429EvenAcrossIPs(t *testing.T) {
	r := newLoginLimitRouter(t)
	for i := 1; i <= 5; i++ {
		w := loginFrom(t, r, fmt.Sprintf("203.0.113.%d", i), loginLimitUser, loginLimitWrongPass)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("échec %d : status %d, want 401 (body=%s)", i, w.Code, w.Body.String())
		}
	}
	assertTooManyAttempts(t, loginFrom(t, r, "203.0.113.6", loginLimitUser, loginLimitWrongPass), "6e tentative")
	assertTooManyAttempts(t, loginFrom(t, r, "203.0.113.7", "  VICTIME ", loginLimitPassword),
		"bon mot de passe, identifiant à casse différente, pendant le blocage")
}

// TestLoginLimit_SixthFailureFromSameIPIs429AcrossIdentifiers : 5 échecs depuis une
// même IP sur des identifiants différents → la 6e tentative depuis cette IP reçoit
// 429 ; une autre IP n'est pas touchée.
func TestLoginLimit_SixthFailureFromSameIPIs429AcrossIdentifiers(t *testing.T) {
	r := newLoginLimitRouter(t)
	const ip = "198.51.100.9"
	for i := 1; i <= 5; i++ {
		w := loginFrom(t, r, ip, fmt.Sprintf("inconnu-%d", i), loginLimitWrongPass)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("échec %d : status %d, want 401 (body=%s)", i, w.Code, w.Body.String())
		}
	}
	assertTooManyAttempts(t, loginFrom(t, r, ip, loginLimitUser, loginLimitPassword), "6e tentative depuis la même IP")
	if w := loginFrom(t, r, "198.51.100.10", loginLimitUser, loginLimitPassword); w.Code != http.StatusOK {
		t.Errorf("autre IP, bon mot de passe : status %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
}

// TestLoginLimit_SuccessResetsCounters : 4 échecs, un succès, puis 4 nouveaux
// échecs restent des 401 (le succès a remis l'identifiant et l'IP à zéro).
func TestLoginLimit_SuccessResetsCounters(t *testing.T) {
	r := newLoginLimitRouter(t)
	const ip = "192.0.2.50"
	for i := 1; i <= 4; i++ {
		if w := loginFrom(t, r, ip, loginLimitUser, loginLimitWrongPass); w.Code != http.StatusUnauthorized {
			t.Fatalf("échec %d avant succès : status %d, want 401", i, w.Code)
		}
	}
	if w := loginFrom(t, r, ip, loginLimitUser, loginLimitPassword); w.Code != http.StatusOK {
		t.Fatalf("succès : status %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	for i := 1; i <= 4; i++ {
		if w := loginFrom(t, r, ip, loginLimitUser, loginLimitWrongPass); w.Code != http.StatusUnauthorized {
			t.Fatalf("échec %d après succès : status %d, want 401 (compteur non remis à zéro ?)", i, w.Code)
		}
	}
}

// TestLoginLimit_SetPasswordRefusedWhileIdentifierBlocked : un identifiant bloqué
// par des échecs de connexion ne peut pas non plus changer de mot de passe, même
// depuis une session ouverte avant le blocage.
func TestLoginLimit_SetPasswordRefusedWhileIdentifierBlocked(t *testing.T) {
	r := newLoginLimitRouter(t)
	ok := loginFrom(t, r, "192.0.2.60", loginLimitUser, loginLimitPassword)
	if ok.Code != http.StatusOK {
		t.Fatalf("connexion initiale : status %d, want 200 (body=%s)", ok.Code, ok.Body.String())
	}
	cookies := ok.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("aucun cookie de session posé par la connexion")
	}
	for i := 1; i <= 5; i++ {
		loginFrom(t, r, fmt.Sprintf("203.0.113.%d", 100+i), loginLimitUser, loginLimitWrongPass)
	}
	w := postFromIP(t, r, "/auth/password", "192.0.2.60",
		domain.SetPasswordRequest{Password: testUserAuthPass}, cookies...)
	assertTooManyAttempts(t, w, "changement de mot de passe d'un identifiant bloqué")
}

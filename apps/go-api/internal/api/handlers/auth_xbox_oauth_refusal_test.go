// Package handlers_test — auth_xbox_oauth_refusal_test.go : un refus du callback OAuth
// ramène le navigateur sur l'écran de connexion avec un code d'erreur
// (/login?oauth_error=<code>), jamais un corps JSON brut : le callback est une navigation
// plein écran, pas un appel du front. Cas couverts par auth_xbox_oauth_test.go.
package handlers_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/session"
)

// assertLoginRedirect vérifie une redirection 302 vers /login?oauth_error=<code>.
func assertLoginRedirect(t *testing.T, w *httptest.ResponseRecorder, code string) {
	t.Helper()
	if w.Code != http.StatusFound {
		t.Fatalf("statut %d, attendu 302 vers l'écran de connexion (%s)", w.Code, w.Body.String())
	}
	loc, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatalf("Location illisible : %v", err)
	}
	if loc.Path != "/login" || loc.Query().Get("oauth_error") != code {
		t.Fatalf("Location = %q, attendu /login?oauth_error=%s", w.Header().Get("Location"), code)
	}
}

// L'échange du code est simulé (aucun appel à Microsoft) : le callback hébergé échoue ensuite à
// l'échange Halo avec la cause donnée. Une panne des services ramène sur /login?oauth_error=halo_unreachable,
// un refus du compte sur halo_exchange_failed.
func TestXboxOAuth_Callback_EchecDeLEchangeHalo_CodeSelonLaCause(t *testing.T) {
	for _, tc := range []struct {
		nom  string
		err  error
		want string
	}{
		{"délai dépassé", fmt.Errorf("sisu_provider: échange XBL: %w", context.DeadlineExceeded), "halo_unreachable"},
		{"erreur réseau", fmt.Errorf("sisu_provider: échange XSTS: %w", &net.OpError{Op: "dial", Err: errors.New("connection refused")}), "halo_unreachable"},
		{"panne 503 des services", errors.New("XSTS token: HTTP 503 depuis https://xsts/x: indisponible"), "halo_unreachable"},
		{"compte refusé 403", errors.New("XSTS token: HTTP 403 depuis https://xsts/x: XErr 2148916233"), "halo_exchange_failed"},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			sessStore := session.NewStore(filepath.Join(t.TempDir(), "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
			h := handlers.NewXboxOAuthHandler(sessStore, &stubTokenProvider{exchangeErr: tc.err}, false, "http://localhost:8000/cb").
				WithCodeExchange(func(context.Context, string, string, string) (*auth_platform.AuthCodeResult, error) {
					return &auth_platform.AuthCodeResult{AccessToken: "ms-access", RefreshToken: "ms-refresh"}, nil
				})
			r := chi.NewRouter()
			r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
			r.Get("/auth/xbox/login", h.LoginRedirect)
			r.Get("/auth/xbox/callback", h.Callback)

			loginW := httptest.NewRecorder()
			r.ServeHTTP(loginW, httptest.NewRequest(http.MethodGet, "/auth/xbox/login", nil))
			loc, err := url.Parse(loginW.Header().Get("Location"))
			if err != nil || loginW.Code != http.StatusFound {
				t.Fatalf("LoginRedirect : %d %q", loginW.Code, loginW.Header().Get("Location"))
			}
			cb := httptest.NewRequest(http.MethodGet, "/auth/xbox/callback?code=code-ok&state="+url.QueryEscape(loc.Query().Get("state")), nil)
			for _, c := range loginW.Result().Cookies() {
				cb.AddCookie(c)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, cb)
			assertLoginRedirect(t, w, tc.want)
			if strings.Contains(w.Header().Get("Location"), "XErr") {
				t.Errorf("le détail de la cause ne doit pas fuiter dans l'adresse : %q", w.Header().Get("Location"))
			}
		})
	}
}

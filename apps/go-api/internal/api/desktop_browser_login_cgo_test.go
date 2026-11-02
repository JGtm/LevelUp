//go:build cgo

// desktop_browser_login_cgo_test.go — la connexion par le navigateur (L21.00) sur la pile réelle du
// routeur desktop : le démarrage derrière le cookie de la coquille, le retour de Microsoft ouvert
// sans cookie mais jamais ailleurs, et rien de monté hors du profil. Le flux complet avec un Azure
// simulé est dans handlers/auth_browser_login_test.go.

package api_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const browserLoginCallbackHost = "localhost:49555"

func TestDesktopBrowserLogin_DemarrageDerriereLeCookieEtURLMicrosoft(t *testing.T) {
	h := buildDesktopRouter(t)
	if rec := (desktopReq{target: "/api/v1/auth/xbox/login"}).do(h); rec.Code != http.StatusUnauthorized {
		t.Errorf("démarrage sans cookie : %d, attendu 401", rec.Code)
	}

	cookie := enterDesktop(t, h)
	rec := (desktopReq{target: "/api/v1/auth/xbox/login", cookie: cookie, locale: "en"}).do(h)
	var out struct {
		AttemptID    string `json:"attempt_id"`
		AuthorizeURL string `json:"authorize_url"`
	}
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &out) != nil {
		t.Fatalf("démarrage : %d %s", rec.Code, rec.Body.String())
	}
	u, err := url.Parse(out.AuthorizeURL)
	if err != nil || u.Host != "login.microsoftonline.com" {
		t.Fatalf("authorize_url = %q (%v), attendu une adresse login.microsoftonline.com", out.AuthorizeURL, err)
	}
	q := u.Query()
	if got, want := q.Get("redirect_uri"), "http://localhost:49555/auth/xbox/callback"; got != want {
		t.Errorf("redirect_uri = %q, attendu %q (localhost, port lié)", got, want)
	}
	if q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "" || len(q.Get("state")) < 32 || q.Get("response_type") != "code" {
		t.Errorf("paramètres d'autorisation inattendus : %v", q)
	}
	if out.AttemptID == "" || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("attempt_id %q, Cache-Control %q", out.AttemptID, rec.Header().Get("Cache-Control"))
	}
}

func TestDesktopBrowserLogin_RetourOuvertSansCookieSeulementPourCetteRoute(t *testing.T) {
	h := buildDesktopRouter(t)

	// Sans cookie, Host localhost : le handler répond (state inconnu = page d'échec 400), pas le garde.
	rec := (desktopReq{target: "/auth/xbox/callback?code=c&state=inconnu", host: browserLoginCallbackHost}).do(h)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "state_invalid") {
		t.Errorf("retour sans cookie : %d %q, attendu la page d'échec state_invalid", rec.Code, rec.Body.String())
	}
	if !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
		t.Errorf("Content-Type = %q, attendu une page HTML", rec.Header().Get("Content-Type"))
	}

	refus := []struct {
		nom string
		req desktopReq
		now int
	}{
		{"autre chemin, Host localhost", desktopReq{target: "/api/v1/bootstrap", host: browserLoginCallbackHost}, http.StatusMisdirectedRequest},
		{"Host étranger", desktopReq{target: "/auth/xbox/callback?state=x", host: "evil.example:49555"}, http.StatusMisdirectedRequest},
		{"autre port", desktopReq{target: "/auth/xbox/callback?state=x", host: "localhost:1"}, http.StatusMisdirectedRequest},
		{"pair hors boucle locale", desktopReq{target: "/auth/xbox/callback?state=x", host: browserLoginCallbackHost, peer: "10.0.0.5:4000"}, http.StatusForbidden},
		{"POST", desktopReq{method: http.MethodPost, target: "/auth/xbox/callback?state=x", host: browserLoginCallbackHost}, http.StatusMisdirectedRequest},
	}
	for _, c := range refus {
		if rec := c.req.do(h); rec.Code != c.now {
			t.Errorf("%s : %d, attendu %d", c.nom, rec.Code, c.now)
		}
	}
}

func TestDesktopBrowserLogin_AbsentHorsDuProfilDesktop(t *testing.T) {
	h := buildTestRouter(t)
	for _, target := range []string{"/auth/xbox/callback?code=c&state=s", "/api/v1/auth/xbox/login"} {
		rec := (desktopReq{target: target, host: "localhost:8000"}).do(h)
		if strings.Contains(rec.Body.String(), "state_invalid") || strings.Contains(rec.Body.String(), "authorize_url") {
			t.Errorf("GET %s hors desktop : %d %q, la connexion par le navigateur ne doit pas y être montée", target, rec.Code, rec.Body.String())
		}
	}
}

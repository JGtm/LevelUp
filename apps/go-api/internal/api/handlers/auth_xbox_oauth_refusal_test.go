// Package handlers_test — auth_xbox_oauth_refusal_test.go : un refus du callback OAuth
// ramène le navigateur sur l'écran de connexion avec un code d'erreur
// (/login?oauth_error=<code>), jamais un corps JSON brut : le callback est une navigation
// plein écran, pas un appel du front. Cas couverts par auth_xbox_oauth_test.go.
package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
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

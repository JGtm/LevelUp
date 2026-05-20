// Package auth — auth_code_test.go : tests ExchangeAuthorizationCode + BuildAuthorizeURL.
package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestExchangeAuthorizationCode_EmptyCode(t *testing.T) {
	_, err := ExchangeAuthorizationCode(context.Background(), "", "https://example.com/cb")
	if err == nil {
		t.Fatal("attendu erreur pour code vide")
	}
	if !strings.Contains(err.Error(), "code vide") {
		t.Errorf("err = %v, want contient 'code vide'", err)
	}
}

func TestExchangeAuthorizationCode_EmptyRedirectURI(t *testing.T) {
	_, err := ExchangeAuthorizationCode(context.Background(), "test-code", "")
	if err == nil {
		t.Fatal("attendu erreur pour redirect_uri vide")
	}
	if !strings.Contains(err.Error(), "redirect_uri vide") {
		t.Errorf("err = %v, want contient 'redirect_uri vide'", err)
	}
}

func TestBuildAuthorizeURL_Format(t *testing.T) {
	redirectURI := "http://localhost:8000/api/v1/auth/xbox/callback"
	state := "abc123def456"

	got := BuildAuthorizeURL(redirectURI, state)

	// Doit pointer vers Microsoft consumers /authorize.
	if !strings.HasPrefix(got, MSALAuthority+"/oauth2/v2.0/authorize?") {
		t.Errorf("URL ne commence pas par l'authority attendue, got: %s", got)
	}

	// Parser pour vérifier les paramètres clés.
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("URL invalide: %v", err)
	}
	q := u.Query()
	if q.Get("response_type") != "code" {
		t.Errorf("response_type = %q, want 'code'", q.Get("response_type"))
	}
	if q.Get("redirect_uri") != redirectURI {
		t.Errorf("redirect_uri = %q, want %q", q.Get("redirect_uri"), redirectURI)
	}
	if q.Get("state") != state {
		t.Errorf("state = %q, want %q", q.Get("state"), state)
	}
	if !strings.Contains(q.Get("scope"), "Xboxlive.signin") {
		t.Errorf("scope ne contient pas Xboxlive.signin: %q", q.Get("scope"))
	}
	if q.Get("client_id") == "" {
		t.Error("client_id manquant")
	}
}

// TestExchangeAuthorizationCode_LiveServer teste contre un serveur HTTP local
// qui simule la réponse Microsoft. Validation de la sérialisation des paramètres
// + parsing de la réponse.
//
// Note : cette fonction n'override pas msalTokenURL (const) ; on teste à la
// place le helper indirect via les conditions invariantes. Pour un vrai test
// d'intégration, il faudrait factoriser le HTTP client en dépendance injectable.
// Trade-off : tests unitaires couvrent les invariants ; tests d'intégration
// (manuels avec Azure) couvrent le round-trip.
func TestExchangeAuthorizationCode_MockServer_Skipped(t *testing.T) {
	// Démontre la structure d'un test intégré qu'on activerait si msalTokenURL
	// devenait paramétrable. Pour PR 4, on accepte ce gap et on couvre via
	// le test du handler Callback qui simule le flow complet.
	if testing.Short() {
		t.Skip("test placeholder pour ExchangeAuthorizationCode avec mock HTTP")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Vérifie les paramètres reçus.
		if err := r.ParseForm(); err != nil {
			http.Error(w, "parse form", 500)
			return
		}
		if r.Form.Get("grant_type") != "authorization_code" {
			http.Error(w, "bad grant_type", 400)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "mock-access-token",
			"refresh_token": "mock-refresh-token",
			"expires_in":    3600,
		})
	}))
	defer server.Close()

	// Sans pouvoir réécrire msalTokenURL, on documente le test comme placeholder.
	t.Skip("msalTokenURL non paramétrable — test à activer si l'API est ouverte")
}

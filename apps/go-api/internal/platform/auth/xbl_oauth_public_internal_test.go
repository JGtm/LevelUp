// Package auth — xbl_oauth_public_internal_test.go : le client PUBLIC de la connexion par le
// navigateur du poste de bureau (ADR 0045 D-19, D-22). Test interne pour overrider msalTokenURL.
package auth

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

// TestExchangeAuthorizationCodePublic_JamaisDeSecret : même avec un secret et un identifiant posés
// dans l'environnement (le `.env.local` d'un dépôt chargé par un serveur de bureau), la requête
// /token est celle d'un client public : l'identifiant fixe de l'application partagée, aucun secret.
func TestExchangeAuthorizationCodePublic_JamaisDeSecret(t *testing.T) {
	for _, tc := range []struct{ nom, secret, clientID string }{
		{"environnement vierge", "", ""},
		{"secret et identifiant dans l'environnement", "s3cret-du-depot", "autre-app-id"},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			var form url.Values
			calls := 0
			withMockTokenEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				_ = r.ParseForm()
				form = r.PostForm
				_, _ = w.Write([]byte(`{"access_token":"at-1","refresh_token":"rt-1","expires_in":3600}`))
			}, tc.clientID, tc.secret)

			res, err := ExchangeAuthorizationCodePublic(context.Background(), "code-1", "http://localhost:51234/auth/xbox/callback", "verifier-1")
			if err != nil || res.RefreshToken != "rt-1" {
				t.Fatalf("échange = (%+v, %v)", res, err)
			}
			if calls != 1 {
				t.Fatalf("appels /token = %d, attendu 1", calls)
			}
			if _, present := form["client_secret"]; present {
				t.Errorf("client_secret présent dans la requête /token d'un client public : %v", form)
			}
			if got := form.Get("client_id"); got != LevelUpClientID {
				t.Errorf("client_id = %q, attendu l'application partagée %q", got, LevelUpClientID)
			}
			if form.Get("code_verifier") != "verifier-1" || form.Get("grant_type") != "authorization_code" {
				t.Errorf("PKCE ou grant manquant : %v", form)
			}
		})
	}
}

// TestBuildAuthorizeURLPublic_IdentifiantFixe : l'adresse /authorize ignore LEVELUP_OAUTH_CLIENT_ID.
func TestBuildAuthorizeURLPublic_IdentifiantFixe(t *testing.T) {
	t.Setenv("LEVELUP_OAUTH_CLIENT_ID", "autre-app-id")
	u, err := url.Parse(BuildAuthorizeURLPublic("http://localhost:1/auth/xbox/callback", "st", "ch"))
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if q.Get("client_id") != LevelUpClientID {
		t.Errorf("client_id = %q, attendu %q", q.Get("client_id"), LevelUpClientID)
	}
	if q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") != "ch" {
		t.Errorf("PKCE absent : %v", q)
	}
}

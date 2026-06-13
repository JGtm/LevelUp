// Package auth — oauth_refresh_internal_test.go : tests internes de l'échange
// OAuth v2 (retry public AADSTS90023, erreurs typées). Test interne (package
// auth) pour pouvoir overrider msalTokenURL vers un httptest.Server.
package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// withMockTokenEndpoint redirige msalTokenURL vers un httptest.Server le temps
// du test, et pose les env vars client_id/secret.
func withMockTokenEndpoint(t *testing.T, handler http.HandlerFunc, clientID, secret string) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	prev := msalTokenURL
	msalTokenURL = srv.URL
	t.Cleanup(func() { msalTokenURL = prev })
	t.Setenv("LEVELUP_OAUTH_CLIENT_ID", clientID)
	t.Setenv("SPNKR_AZURE_CLIENT_SECRET", secret)
}

const aadsts90023JSON = `{"error":"invalid_request","error_description":"AADSTS90023: Public clients can't send a client secret. Trace ID: x"}`

// TestExchangeRefreshToken_SecretAccepted : app confidentielle, le secret passe
// en 1re tentative, pas de retry.
func TestExchangeRefreshToken_SecretAccepted(t *testing.T) {
	calls := 0
	withMockTokenEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = r.ParseForm()
		if r.PostForm.Get("client_secret") == "" {
			t.Errorf("1re tentative : client_secret attendu")
		}
		_, _ = w.Write([]byte(`{"access_token":"at-1","refresh_token":"rt-2","expires_in":3600}`))
	}, "custom-app-id", "s3cret")

	at, rt, err := ExchangeRefreshTokenWithRotation(context.Background(), "rt-1")
	if err != nil {
		t.Fatalf("err inattendue : %v", err)
	}
	if at != "at-1" || rt != "rt-2" {
		t.Errorf("tokens : obtenu (%q, %q), attendu (at-1, rt-2)", at, rt)
	}
	if calls != 1 {
		t.Errorf("appels endpoint : attendu 1, obtenu %d", calls)
	}
}

// TestExchangeRefreshToken_RetryPublicOnAADSTS90023 : la 1re tentative avec
// secret est refusée (90023), le retry sans secret réussit.
func TestExchangeRefreshToken_RetryPublicOnAADSTS90023(t *testing.T) {
	calls := 0
	withMockTokenEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = r.ParseForm()
		if r.PostForm.Get("client_secret") != "" {
			_, _ = w.Write([]byte(aadsts90023JSON))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"at-pub","refresh_token":"rt-pub","expires_in":3600}`))
	}, "custom-app-id", "s3cret")

	at, rt, err := ExchangeRefreshTokenWithRotation(context.Background(), "rt-1")
	if err != nil {
		t.Fatalf("err inattendue : %v", err)
	}
	if at != "at-pub" || rt != "rt-pub" {
		t.Errorf("tokens : obtenu (%q, %q), attendu (at-pub, rt-pub)", at, rt)
	}
	if calls != 2 {
		t.Errorf("appels endpoint : attendu 2 (avec secret puis sans), obtenu %d", calls)
	}
}

// TestExchangeRefreshToken_RetryPublicThenRevoked : le retry public échoue en
// invalid_grant → erreur typée classe Revoked.
func TestExchangeRefreshToken_RetryPublicThenRevoked(t *testing.T) {
	withMockTokenEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostForm.Get("client_secret") != "" {
			_, _ = w.Write([]byte(aadsts90023JSON))
			return
		}
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"AADSTS70000: token expired"}`))
	}, "custom-app-id", "s3cret")

	_, _, err := ExchangeRefreshTokenWithRotation(context.Background(), "rt-dead")
	if err == nil {
		t.Fatalf("erreur attendue")
	}
	var oerr *OAuthExchangeError
	if !errors.As(err, &oerr) {
		t.Fatalf("erreur typée OAuthExchangeError attendue, obtenu %T", err)
	}
	if got := ClassifyAuthError(err); got != AuthErrorRevoked {
		t.Errorf("classe : attendu revoked, obtenu %s", got)
	}
}

// TestExchangeRefreshToken_NoSecretNoRetry : sans secret env, jamais de
// client_secret dans la requête et aucun retry sur erreur.
func TestExchangeRefreshToken_NoSecretNoRetry(t *testing.T) {
	calls := 0
	withMockTokenEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = r.ParseForm()
		if r.PostForm.Get("client_secret") != "" {
			t.Errorf("client_secret ne doit pas être envoyé sans env secret")
		}
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"dead"}`))
	}, "custom-app-id", "")

	_, _, err := ExchangeRefreshTokenWithRotation(context.Background(), "rt-1")
	if err == nil {
		t.Fatalf("erreur attendue")
	}
	if calls != 1 {
		t.Errorf("appels endpoint : attendu 1 (pas de retry), obtenu %d", calls)
	}
}

// TestExchangeRefreshToken_CanonicalClientSendsSecret : Phase 3 — avec LevelUpClientID
// (e1cb35ab, redirect Web = confidentiel), le secret EST envoyé (AVANT il était exclu).
func TestExchangeRefreshToken_CanonicalClientSendsSecret(t *testing.T) {
	withMockTokenEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostForm.Get("client_secret") == "" {
			t.Errorf("client_secret attendu pour e1cb35ab (confidentiel)")
		}
		_, _ = w.Write([]byte(`{"access_token":"at","expires_in":3600}`))
	}, "", "s3cret") // LEVELUP_OAUTH_CLIENT_ID vide → LevelUpClientID (e1cb35ab)

	if _, _, err := ExchangeRefreshTokenWithRotation(context.Background(), "rt-1"); err != nil {
		t.Fatalf("err inattendue : %v", err)
	}
}

// TestOAuthExchangeError_Class : classification des codes serveur.
func TestOAuthExchangeError_Class(t *testing.T) {
	tests := []struct {
		code string
		want AuthErrorClass
	}{
		{"invalid_grant", AuthErrorRevoked},
		{"interaction_required", AuthErrorRevoked},
		{"invalid_request", AuthErrorConfig},
		{"invalid_client", AuthErrorConfig},
		{"unauthorized_client", AuthErrorConfig},
		{"temporarily_unavailable", AuthErrorTransient},
	}
	for _, tc := range tests {
		e := &OAuthExchangeError{ErrorCode: tc.code}
		if got := e.Class(); got != tc.want {
			t.Errorf("Class(%s) = %s, attendu %s", tc.code, got, tc.want)
		}
	}
	// Erreur non typée (réseau) → transient.
	if got := ClassifyAuthError(errors.New("dial tcp: timeout")); got != AuthErrorTransient {
		t.Errorf("erreur non typée : attendu transient, obtenu %s", got)
	}
}

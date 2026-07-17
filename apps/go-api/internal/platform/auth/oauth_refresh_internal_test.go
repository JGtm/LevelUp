// Package auth — oauth_refresh_internal_test.go : tests internes de l'échange
// OAuth v2 (retry public AADSTS90023, erreurs typées). Test interne (package
// auth) pour pouvoir overrider msalTokenURL vers un httptest.Server.
package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// withMockTokenEndpoint redirige msalTokenURL vers un httptest.Server le temps
// du test, et pose les env vars client_id/secret. Stub aussi msaTokenURL (mort
// par défaut) : un invalid_grant Azure déclenche le fallback MSA natif — sans
// stub, les tests taperaient le vrai login.live.com.
func withMockTokenEndpoint(t *testing.T, handler http.HandlerFunc, clientID, secret string) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	prev := msalTokenURL
	msalTokenURL = srv.URL
	t.Cleanup(func() { msalTokenURL = prev })
	withMockMSAEndpoint(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"RT inconnu du client MSA (stub test)"}`))
	})
	t.Setenv("LEVELUP_OAUTH_CLIENT_ID", clientID)
	t.Setenv("SPNKR_AZURE_CLIENT_SECRET", secret)
}

// withMockMSAEndpoint redirige msaTokenURL (fallback MSA natif SISU) vers un
// httptest.Server le temps du test.
func withMockMSAEndpoint(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	prev := msaTokenURL
	msaTokenURL = srv.URL
	t.Cleanup(func() { msaTokenURL = prev })
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

	at, rt, family, err := ExchangeRefreshTokenWithRotation(context.Background(), "rt-1")
	if err != nil {
		t.Fatalf("err inattendue : %v", err)
	}
	if at != "at-1" || rt != "rt-2" {
		t.Errorf("tokens : obtenu (%q, %q), attendu (at-1, rt-2)", at, rt)
	}
	if family != TokenFamilyAzure {
		t.Errorf("provenance (F12) : obtenu %q, attendu %q (app Azure)", family, TokenFamilyAzure)
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

	at, rt, _, err := ExchangeRefreshTokenWithRotation(context.Background(), "rt-1")
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

	_, _, _, err := ExchangeRefreshTokenWithRotation(context.Background(), "rt-dead")
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

	_, _, _, err := ExchangeRefreshTokenWithRotation(context.Background(), "rt-1")
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

	if _, _, _, err := ExchangeRefreshTokenWithRotation(context.Background(), "rt-1"); err != nil {
		t.Fatalf("err inattendue : %v", err)
	}
}

// TestExchangeRefreshToken_FallbackMSAOnInvalidGrant : un RT émis par le client
// Xbox natif (device-flow SISU) est refusé en invalid_grant par l'endpoint Azure
// v2 → le fallback MSA natif (login.live.com, client Xbox, scope MBI_SSL, sans
// secret) doit prendre le relais et réussir.
func TestExchangeRefreshToken_FallbackMSAOnInvalidGrant(t *testing.T) {
	withMockTokenEndpoint(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"AADSTS70000: grant issued to a different client"}`))
	}, "custom-app-id", "")

	msaCalls := 0
	withMockMSAEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		msaCalls++
		_ = r.ParseForm()
		if got := r.PostForm.Get("client_id"); got != SISUDefaultAppID {
			t.Errorf("client_id MSA : attendu %q (client Xbox), obtenu %q", SISUDefaultAppID, got)
		}
		if got := r.PostForm.Get("scope"); got != sisuMSAScope {
			t.Errorf("scope MSA : attendu %q, obtenu %q", sisuMSAScope, got)
		}
		if r.PostForm.Get("client_secret") != "" {
			t.Errorf("client_secret interdit sur le flux MSA natif (client public)")
		}
		_, _ = w.Write([]byte(`{"access_token":"at-msa","refresh_token":"rt-msa-2","expires_in":86400}`))
	})

	at, rt, family, err := ExchangeRefreshTokenWithRotation(context.Background(), "rt-msa-1")
	if err != nil {
		t.Fatalf("err inattendue : %v", err)
	}
	if at != "at-msa" || rt != "rt-msa-2" {
		t.Errorf("tokens : obtenu (%q, %q), attendu (at-msa, rt-msa-2)", at, rt)
	}
	if family != TokenFamilyXboxNative {
		t.Errorf("provenance (F12) : obtenu %q, attendu %q (MSA natif/Xbox)", family, TokenFamilyXboxNative)
	}
	if msaCalls != 1 {
		t.Errorf("appels endpoint MSA : attendu 1, obtenu %d", msaCalls)
	}
}

// TestExchangeRefreshToken_FallbackMSAAlsoDead : Azure ET MSA refusent le RT →
// l'erreur Azure INITIALE est propagée (classification revoked intacte pour le
// pool/resolver).
func TestExchangeRefreshToken_FallbackMSAAlsoDead(t *testing.T) {
	withMockTokenEndpoint(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"AADSTS70000: token expired"}`))
	}, "custom-app-id", "")
	// Le stub MSA par défaut de withMockTokenEndpoint répond déjà invalid_grant.

	_, _, _, err := ExchangeRefreshTokenWithRotation(context.Background(), "rt-dead")
	if err == nil {
		t.Fatalf("erreur attendue")
	}
	var oerr *OAuthExchangeError
	if !errors.As(err, &oerr) {
		t.Fatalf("erreur typée OAuthExchangeError attendue, obtenu %T", err)
	}
	if !strings.Contains(oerr.Description, "AADSTS70000") {
		t.Errorf("l'erreur Azure initiale doit être propagée, obtenu %q", oerr.Description)
	}
	if got := ClassifyAuthError(err); got != AuthErrorRevoked {
		t.Errorf("classe : attendu revoked, obtenu %s", got)
	}
}

// TestExchangeRefreshToken_NoFallbackOnConfigError : une erreur de classe config
// (invalid_client…) ne déclenche PAS le fallback MSA.
func TestExchangeRefreshToken_NoFallbackOnConfigError(t *testing.T) {
	withMockTokenEndpoint(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":"invalid_client","error_description":"app inconnue"}`))
	}, "custom-app-id", "")

	msaCalls := 0
	withMockMSAEndpoint(t, func(w http.ResponseWriter, _ *http.Request) {
		msaCalls++
		_, _ = w.Write([]byte(`{"access_token":"at-msa"}`))
	})

	_, _, _, err := ExchangeRefreshTokenWithRotation(context.Background(), "rt-1")
	if err == nil {
		t.Fatalf("erreur attendue")
	}
	if msaCalls != 0 {
		t.Errorf("le fallback MSA ne doit pas être tenté sur une erreur config, obtenu %d appels", msaCalls)
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

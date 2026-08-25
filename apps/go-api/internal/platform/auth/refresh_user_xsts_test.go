// Package auth — refresh_user_xsts_test.go : tests RefreshUserXSTS.
//
// Couvre les invariants et chemins d'erreur. Le succès complet (AcquireXSTSForRTA)
// dépend d'un appel HTTP réel vers Xbox Live et n'est pas testé unitairement
// (couvert par les tests d'intégration manuels).
package auth

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestMultiUserStore(t *testing.T) *MultiUserTokenStore {
	t.Helper()
	return NewMultiUserTokenStore(filepath.Join(t.TempDir(), "watcher_tokens"))
}

func TestRefreshUserXSTS_UserNotFound(t *testing.T) {
	store := newTestMultiUserStore(t)

	_, err := RefreshUserXSTS(context.Background(), store, "99999999")
	if err == nil {
		t.Fatal("attendu erreur pour xuid inconnu")
	}
	if !errors.Is(err, ErrUserTokensNotFound) && !strings.Contains(err.Error(), "load") {
		t.Errorf("err = %v, want wrapping ErrUserTokensNotFound ou contenant 'load'", err)
	}
}

func TestRefreshUserXSTS_NoCacheNoValidAccessToken(t *testing.T) {
	store := newTestMultiUserStore(t)
	// User persisté sans MSAL cache + access_token expiré.
	_ = store.Upsert(&UserTokens{
		XUID:           "12345",
		Gamertag:       "TestUser",
		AccessToken:    "expired-token",
		OAuthExpiresAt: time.Now().Add(-1 * time.Hour),
	})

	_, err := RefreshUserXSTS(context.Background(), store, "12345")
	if err == nil {
		t.Fatal("attendu erreur quand aucun access_token utilisable")
	}
	if !strings.Contains(err.Error(), "access_token") {
		t.Errorf("err = %v, want contient 'access_token'", err)
	}
}

func TestRefreshUserXSTS_AccessTokenValid_TriesXSTSAcquire(t *testing.T) {
	store := newTestMultiUserStore(t)
	// User avec access_token encore valide → on appellera AcquireXSTSForRTA
	// avec ce token. L'appel réseau échouera (pas de vrai Microsoft) mais le
	// flow a bien tenté → preuve que refresh_access_token a réutilisé le token.
	_ = store.Upsert(&UserTokens{
		XUID:           "12345",
		Gamertag:       "TestUser",
		AccessToken:    "valid-but-not-real-token",
		OAuthExpiresAt: time.Now().Add(50 * time.Minute),
	})

	_, err := RefreshUserXSTS(context.Background(), store, "12345")
	if err == nil {
		t.Fatal("attendu erreur (AcquireXSTSForRTA échoue avec token bidon)")
	}
	// L'erreur doit venir de AcquireXSTSForRTA, pas du refresh access_token.
	// On vérifie que le message mentionne XSTS, pas "access_token".
	if strings.Contains(err.Error(), "aucun access_token") {
		t.Errorf("err = %v, ne devrait pas dire 'aucun access_token' (le stocké était valide)", err)
	}
}

func TestRefreshUserXSTS_InvalidXUID(t *testing.T) {
	store := newTestMultiUserStore(t)

	// XUID unsafe (contient des caractères non-numériques) → load échoue.
	_, err := RefreshUserXSTS(context.Background(), store, "../escape")
	if err == nil {
		t.Fatal("attendu erreur pour xuid invalide")
	}
}

func TestRefreshAccessTokenForUser_NoCache_AccessTokenStillValid(t *testing.T) {
	// Test du helper interne : pas de cache MSAL mais access_token encore valide
	// → réutilisé tel quel.
	tokens := &UserTokens{
		XUID:           "12345",
		AccessToken:    "still-valid",
		OAuthExpiresAt: time.Now().Add(30 * time.Minute),
	}
	got := refreshAccessTokenForUser(context.Background(), nil, tokens)
	if got != "still-valid" {
		t.Errorf("got = %q, want still-valid (réutilise stocké)", got)
	}
}

func TestRefreshAccessTokenForUser_NoCache_AccessTokenExpired(t *testing.T) {
	tokens := &UserTokens{
		XUID:           "12345",
		AccessToken:    "expired",
		OAuthExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	got := refreshAccessTokenForUser(context.Background(), nil, tokens)
	if got != "" {
		t.Errorf("got = %q, want vide (token expiré ne devrait pas être réutilisé)", got)
	}
}

func TestRefreshAccessTokenForUser_NoCache_NoToken(t *testing.T) {
	tokens := &UserTokens{
		XUID:           "12345",
		AccessToken:    "",
		OAuthExpiresAt: time.Now().Add(30 * time.Minute),
	}
	got := refreshAccessTokenForUser(context.Background(), nil, tokens)
	if got != "" {
		t.Errorf("got = %q, want vide (pas de token et pas de cache)", got)
	}
}

// TestRefreshUserXSTS_RotatedRTPersistedBeforeXSTS — AU1 (revue 2026-07). Le refresh
// OAuth rote le RT ; l'acquisition XSTS échoue ENSUITE (endpoint Xbox injoignable en
// test). Le RT roté doit DÉJÀ être persisté dans le store — sans le persist immédiat,
// l'échec XSTS retourne avant le store.Upsert final et le store garde le RT consommé
// (invalid_grant au prochain refresh).
func TestRefreshUserXSTS_RotatedRTPersistedBeforeXSTS(t *testing.T) {
	// Endpoint OAuth mocké : refresh réussi avec un RT ROTÉ.
	withMockTokenEndpoint(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"at-fresh","refresh_token":"rt-rotated","expires_in":3600}`))
	}, "app-id", "s3cret")

	store := newTestMultiUserStore(t)
	if err := store.Upsert(&UserTokens{
		XUID:              "12345",
		Gamertag:          "TestUser",
		OAuthRefreshToken: "rt-old",
	}); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	// AcquireXSTSForRTA échoue (pas de vrai Xbox / at bidon) → XSTS transitoire.
	if _, err := RefreshUserXSTS(context.Background(), store, "12345"); err == nil {
		t.Fatal("attendu échec XSTS (endpoint Xbox injoignable en test)")
	}

	reloaded, err := store.Load("12345")
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	if reloaded.OAuthRefreshToken != "rt-rotated" {
		t.Errorf("RT persisté = %q, attendu rt-rotated (persist immédiat avant XSTS, AU1)", reloaded.OAuthRefreshToken)
	}
}

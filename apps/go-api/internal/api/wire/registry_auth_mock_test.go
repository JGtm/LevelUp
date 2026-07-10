package wire

// registry_auth_mock_test.go — démontre le gain du découplage (Axe 3) :
// tryRefreshFromAuthStore dépend de auth.UserTokenStore (interface) et de
// auth.TokenProvider (interface), donc testable avec des mocks en mémoire,
// SANS répertoire de tokens sur disque ni vrai provider MSAL.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/platform/auth"
)

// mockUserTokenStore implémente auth.UserTokenStore en mémoire.
type mockUserTokenStore struct {
	user        *auth.UserTokens
	updatedXUID string
	updatedRT   string
	clearedXUID string // dernier xuid passé à ClearReauthRequired
	clearCalls  int
}

func (m *mockUserTokenStore) Load(string) (*auth.UserTokens, error) { return m.user, nil }
func (m *mockUserTokenStore) UpdateOAuthRefreshToken(xuid, rt string) error {
	m.updatedXUID, m.updatedRT = xuid, rt
	return nil
}
func (m *mockUserTokenStore) ClearReauthRequired(xuid string) error {
	m.clearedXUID = xuid
	m.clearCalls++
	return nil
}

// fakeTokenProvider implémente auth.TokenProvider ; seul le chemin OAuth refresh
// avec rotation est exercé ici.
type fakeTokenProvider struct {
	rotatedRT string
	failOAuth bool // true → TryOAuthRefreshWithRotation renvoie une erreur (RT revoked)
}

func (f *fakeTokenProvider) InitDeviceFlow(context.Context) (auth.DeviceFlow, error) {
	return nil, nil
}
func (f *fakeTokenProvider) TrySilentRefresh(context.Context, string) (string, error) {
	return "", nil
}
func (f *fakeTokenProvider) TryOAuthRefresh(context.Context, string) (string, error) {
	return "", nil
}
func (f *fakeTokenProvider) TryOAuthRefreshWithRotation(context.Context, string) (string, string, error) {
	if f.failOAuth {
		return "", "", errors.New("invalid_grant: refresh_token revoked")
	}
	return "access-token", f.rotatedRT, nil
}
func (f *fakeTokenProvider) Exchange(context.Context, string) (*auth.ExchangeResult, error) {
	return &auth.ExchangeResult{XUID: "x"}, nil
}

// TestTryRefreshFromAuthStore_PersistsRotatedRT : quand Microsoft rotate le
// refresh token, la rotation est persistée via le store injecté — vérifié avec
// un mock, sans DuckDB ni disque (impossible avant le découplage Axe 3).
func TestTryRefreshFromAuthStore_PersistsRotatedRT(t *testing.T) {
	store := &mockUserTokenStore{
		user: &auth.UserTokens{XUID: "x", OAuthRefreshToken: "rt-old"},
	}
	reg := &ServiceRegistry{
		authStore: store,
		provider:  &fakeTokenProvider{rotatedRT: "rt-new"},
	}

	// pdb non utilisé par tryRefreshFromAuthStore (seulement pour le log du caller).
	result := reg.tryRefreshFromAuthStore(context.Background(), nil, "x")
	if result == nil {
		t.Fatal("résultat nil — l'échange aurait dû réussir avec le provider mock")
	}
	if store.updatedRT != "rt-new" || store.updatedXUID != "x" {
		t.Errorf("rotation RT non persistée: updatedXUID=%q updatedRT=%q, want x/rt-new",
			store.updatedXUID, store.updatedRT)
	}
}

// TestTryRefreshFromAuthStore_NoRotation_NoWrite : si le RT n'est pas rotaté
// (provider renvoie le même), aucune écriture au store.
func TestTryRefreshFromAuthStore_NoRotation_NoWrite(t *testing.T) {
	store := &mockUserTokenStore{
		user: &auth.UserTokens{XUID: "x", OAuthRefreshToken: "rt-same"},
	}
	reg := &ServiceRegistry{
		authStore: store,
		provider:  &fakeTokenProvider{rotatedRT: "rt-same"},
	}

	result := reg.tryRefreshFromAuthStore(context.Background(), nil, "x")
	if result == nil {
		t.Fatal("résultat nil inattendu")
	}
	if store.updatedRT != "" {
		t.Errorf("aucune écriture attendue quand le RT est inchangé, got updatedRT=%q", store.updatedRT)
	}
}

// TestTryRefreshFromAuthStore_ClearsReauthOnSuccess : un refresh par-joueur réussi
// efface le flag reauth_required → auto-guérison de la bannière de reconnexion
// (le RT a prouvé qu'il est vivant). C'est le fix du faux positif « connexion Xbox
// expirée » qui restait collé alors que sync/fetch live fonctionnaient.
func TestTryRefreshFromAuthStore_ClearsReauthOnSuccess(t *testing.T) {
	store := &mockUserTokenStore{
		user: &auth.UserTokens{XUID: "x", OAuthRefreshToken: "rt-old", ReauthRequired: true},
	}
	reg := &ServiceRegistry{
		authStore: store,
		provider:  &fakeTokenProvider{rotatedRT: "rt-new"},
	}

	if result := reg.tryRefreshFromAuthStore(context.Background(), nil, "x"); result == nil {
		t.Fatal("résultat nil — l'échange aurait dû réussir avec le provider mock")
	}
	if store.clearCalls != 1 || store.clearedXUID != "x" {
		t.Errorf("clear reauth attendu pour xuid=x, got clearCalls=%d clearedXUID=%q",
			store.clearCalls, store.clearedXUID)
	}
}

// TestTryRefreshFromAuthStore_NoClearOnFailure : un refresh ÉCHOUÉ (RT révoqué)
// NE DOIT PAS effacer le flag — sinon une bannière de reconnexion légitime
// disparaîtrait pour un xuid au RT réellement mort.
func TestTryRefreshFromAuthStore_NoClearOnFailure(t *testing.T) {
	store := &mockUserTokenStore{
		user: &auth.UserTokens{XUID: "x", OAuthRefreshToken: "rt-revoked"},
	}
	reg := &ServiceRegistry{
		authStore: store,
		provider:  &fakeTokenProvider{failOAuth: true},
	}

	if result := reg.tryRefreshFromAuthStore(context.Background(), nil, "x"); result != nil {
		t.Fatal("résultat non-nil inattendu — le refresh aurait dû échouer")
	}
	if store.clearCalls != 0 {
		t.Errorf("aucun clear attendu sur échec de refresh, got clearCalls=%d", store.clearCalls)
	}
}

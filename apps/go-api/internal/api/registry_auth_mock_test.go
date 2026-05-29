package api

// registry_auth_mock_test.go — démontre le gain du découplage (Axe 3) :
// tryRefreshFromAuthStore dépend de auth.UserTokenStore (interface) et de
// auth.TokenProvider (interface), donc testable avec des mocks en mémoire,
// SANS répertoire de tokens sur disque ni vrai provider MSAL.

import (
	"context"
	"testing"

	"levelup/go-api/internal/platform/auth"
)

// mockUserTokenStore implémente auth.UserTokenStore en mémoire.
type mockUserTokenStore struct {
	user        *auth.UserTokens
	updatedXUID string
	updatedRT   string
}

func (m *mockUserTokenStore) Load(string) (*auth.UserTokens, error) { return m.user, nil }
func (m *mockUserTokenStore) UpdateOAuthRefreshToken(xuid, rt string) error {
	m.updatedXUID, m.updatedRT = xuid, rt
	return nil
}

// fakeTokenProvider implémente auth.TokenProvider ; seul le chemin OAuth refresh
// avec rotation est exercé ici.
type fakeTokenProvider struct {
	rotatedRT string
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

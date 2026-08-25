package auth

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// stubProvider est un TokenProvider minimal pour tester EnsureWatcherAccessToken.
// Les methodes non utilisees retournent des valeurs neutres.
type stubProvider struct {
	oauthResp string
	oauthErr  error
	lastCall  string
}

func (s *stubProvider) InitDeviceFlow(_ context.Context) (DeviceFlow, error) {
	return nil, errors.New("not implemented")
}

func (s *stubProvider) TryOAuthRefresh(_ context.Context, refreshToken string) (string, error) {
	s.lastCall = refreshToken
	return s.oauthResp, s.oauthErr
}

func (s *stubProvider) TryOAuthRefreshWithRotation(_ context.Context, refreshToken string) (string, string, error) {
	s.lastCall = refreshToken
	return s.oauthResp, "", s.oauthErr
}

func (s *stubProvider) Exchange(_ context.Context, _ string) (*ExchangeResult, error) {
	return nil, errors.New("not implemented")
}

func newStoreWithTokens(t *testing.T, tokens *StoredTokens) *TokenStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "watcher_tokens.json")
	store := NewTokenStore(path)
	if tokens != nil {
		if err := store.Save(tokens); err != nil {
			t.Fatalf("seed store: %v", err)
		}
	}
	return store
}

// TestEnsureWatcherAccessToken_AccessTokenValid_NoRefresh verifie qu'un
// access_token encore valide (avec marge confortable) est retourne tel quel,
// sans appeler provider.TryOAuthRefreshWithRotation.
func TestEnsureWatcherAccessToken_AccessTokenValid_NoRefresh(t *testing.T) {
	store := newStoreWithTokens(t, &StoredTokens{
		AccessToken:    "still-valid",
		OAuthExpiresAt: time.Now().Add(30 * time.Minute),
	})
	prov := &stubProvider{}

	got, err := EnsureWatcherAccessToken(context.Background(), nil, store, prov, "JGtm")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "still-valid" {
		t.Errorf("got = %q, want %q (access_token courant aurait du etre reutilise)", got, "still-valid")
	}
	if prov.lastCall != "" {
		t.Errorf("refresh appele alors que l'access_token etait valide (refreshToken = %q)", prov.lastCall)
	}
}

// TestEnsureWatcherAccessToken_NoRefreshTokenAvailable : ADR 0023 Phase 5 — sans
// MultiUserTokenStore, il n'existe plus AUCUNE source de refresh_token → ("", nil)
// sans erreur, pour permettre au caller de retomber sur le mode degrade.
func TestEnsureWatcherAccessToken_NoRefreshTokenAvailable(t *testing.T) {
	store := newStoreWithTokens(t, &StoredTokens{
		AccessToken:    "old",
		OAuthExpiresAt: time.Now().Add(-1 * time.Hour),
	})
	prov := &stubProvider{}

	got, err := EnsureWatcherAccessToken(context.Background(), nil, store, prov, "Unknown_Gamertag_999")
	if err != nil {
		t.Errorf("absence de refresh_token doit retourner (\"\", nil) — got err = %v", err)
	}
	if got != "" {
		t.Errorf("absence de refresh_token doit retourner \"\" — got %q", got)
	}
	if prov.lastCall != "" {
		t.Errorf("refresh ne devait pas etre appele sans refresh_token (got %q)", prov.lastCall)
	}
}

// TestEnsureWatcherAccessToken_ProviderRefreshFails verifie que si le refresh
// echoue, on retourne ("", nil) — pas d'erreur — pour permettre au caller de
// continuer en mode degrade (ex: XSTS deja stocke encore valide).
func TestEnsureWatcherAccessToken_ProviderRefreshFails(t *testing.T) {
	multiStore := newMultiStoreForWatcher(t, "2533274858283686", "JGtm", "rt-revoked")
	store := newStoreWithTokens(t, &StoredTokens{
		AccessToken:    "old",
		OAuthExpiresAt: time.Now().Add(-1 * time.Hour),
	})
	prov := &stubProvider{oauthErr: errors.New("refresh_token revoked")}

	got, err := EnsureWatcherAccessToken(context.Background(), multiStore, store, prov, "JGtm")
	if err != nil {
		t.Errorf("erreur de refresh doit etre absorbee (mode degrade) — got err = %v", err)
	}
	if got != "" {
		t.Errorf("erreur de refresh doit retourner \"\" — got %q", got)
	}
}

// TestEnsureWatcherAccessToken_NilArguments verifie les preconditions.
func TestEnsureWatcherAccessToken_NilArguments(t *testing.T) {
	prov := &stubProvider{}
	store := newStoreWithTokens(t, &StoredTokens{})

	if _, err := EnsureWatcherAccessToken(context.Background(), nil, nil, prov, "JGtm"); err == nil {
		t.Error("store watcher nil doit retourner une erreur")
	}
	if _, err := EnsureWatcherAccessToken(context.Background(), nil, store, nil, "JGtm"); err == nil {
		t.Error("provider nil doit retourner une erreur")
	}
}

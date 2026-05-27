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

func (s *stubProvider) TrySilentRefresh(_ context.Context, _ string) (string, error) {
	return "", nil
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
// sans appeler provider.TryOAuthRefresh.
func TestEnsureWatcherAccessToken_AccessTokenValid_NoRefresh(t *testing.T) {
	store := newStoreWithTokens(t, &StoredTokens{
		AccessToken:    "still-valid",
		RefreshToken:   "rt-1",
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
		t.Errorf("TryOAuthRefresh appele alors que l'access_token etait valide (refreshToken = %q)", prov.lastCall)
	}
}

// TestEnsureWatcherAccessToken_AccessTokenExpired_UsesStoreRefresh verifie
// que si l'access_token est expire et qu'un refresh_token est present dans
// le store, il est utilise pour obtenir un nouvel access_token, qui est
// persiste dans le store.
func TestEnsureWatcherAccessToken_AccessTokenExpired_UsesStoreRefresh(t *testing.T) {
	store := newStoreWithTokens(t, &StoredTokens{
		AccessToken:    "old-expired",
		RefreshToken:   "rt-from-store",
		OAuthExpiresAt: time.Now().Add(-1 * time.Hour),
	})
	prov := &stubProvider{oauthResp: "fresh-token"}

	got, err := EnsureWatcherAccessToken(context.Background(), nil, store, prov, "JGtm")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "fresh-token" {
		t.Errorf("got = %q, want fresh-token", got)
	}
	if prov.lastCall != "rt-from-store" {
		t.Errorf("TryOAuthRefresh appele avec %q, want rt-from-store", prov.lastCall)
	}

	persisted, err := store.Load()
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	if persisted.AccessToken != "fresh-token" {
		t.Errorf("access_token non persiste: got %q", persisted.AccessToken)
	}
	if !persisted.IsOAuthValid(time.Minute) {
		t.Error("access_token persiste devrait avoir un OAuthExpiresAt valide (>1 min future)")
	}
}

// TestEnsureWatcherAccessToken_AccessTokenExpired_FallbackToEnvVar verifie
// que si le refresh_token n'est PAS dans le store mais EST dans la variable
// d'environnement SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>, l'env var est utilisee.
// C'est le cas de production le plus frequent : watcher_tokens.json ne
// contient pas de refresh_token (regenere a chaque XSTS-only refresh), mais
// l'utilisateur a configure ses refresh_tokens dans .env.local.
func TestEnsureWatcherAccessToken_AccessTokenExpired_FallbackToEnvVar(t *testing.T) {
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_JGTM", "rt-from-env")

	store := newStoreWithTokens(t, &StoredTokens{
		AccessToken:    "old-expired",
		RefreshToken:   "", // PAS de refresh dans le fichier — typique chez l'user
		OAuthExpiresAt: time.Now().Add(-1 * time.Hour),
	})
	prov := &stubProvider{oauthResp: "fresh-token-from-env"}

	got, err := EnsureWatcherAccessToken(context.Background(), nil, store, prov, "JGtm")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "fresh-token-from-env" {
		t.Errorf("got = %q, want fresh-token-from-env", got)
	}
	if prov.lastCall != "rt-from-env" {
		t.Errorf("TryOAuthRefresh devait recevoir le rt-from-env de l'env var, got %q", prov.lastCall)
	}

	persisted, _ := store.Load()
	if persisted.RefreshToken != "rt-from-env" {
		t.Errorf("le refresh_token de l'env var doit etre persiste dans watcher_tokens.json apres refresh (got %q)", persisted.RefreshToken)
	}
}

// TestEnsureWatcherAccessToken_GamertagNormalization verifie que la cle env
// est construite avec la normalisation attendue (uppercase + remplacement
// des caracteres ' ', '-', '.' par '_').
func TestEnsureWatcherAccessToken_GamertagNormalization(t *testing.T) {
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_MY_USER_PRO", "rt-normalized")

	store := newStoreWithTokens(t, &StoredTokens{
		AccessToken:    "old",
		RefreshToken:   "",
		OAuthExpiresAt: time.Now().Add(-1 * time.Hour),
	})
	prov := &stubProvider{oauthResp: "fresh"}

	// "My-User.Pro" doit donner la cle SPNKR_OAUTH_REFRESH_TOKEN_MY_USER_PRO
	got, err := EnsureWatcherAccessToken(context.Background(), nil, store, prov, "My-User.Pro")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "fresh" {
		t.Errorf("got = %q, want fresh (env var avec gamertag normalise non trouvee)", got)
	}
	if prov.lastCall != "rt-normalized" {
		t.Errorf("normalisation gamertag ratee : TryOAuthRefresh got %q, want rt-normalized", prov.lastCall)
	}
}

// TestEnsureWatcherAccessToken_NoRefreshTokenAvailable verifie qu'en l'absence
// de refresh_token (ni dans le store ni dans l'env), on retourne ("", nil)
// sans erreur, pour permettre au caller de retomber sur le mode degrade.
func TestEnsureWatcherAccessToken_NoRefreshTokenAvailable(t *testing.T) {
	// PAS de t.Setenv → env var absente
	store := newStoreWithTokens(t, &StoredTokens{
		AccessToken:    "old",
		RefreshToken:   "",
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
		t.Errorf("TryOAuthRefresh ne devait pas etre appele sans refresh_token (got %q)", prov.lastCall)
	}
}

// TestEnsureWatcherAccessToken_ProviderRefreshFails verifie que si
// TryOAuthRefresh echoue, on retourne ("", nil) — pas d'erreur — pour permettre
// au caller de continuer en mode degrade (ex: XSTS deja stocke encore valide).
func TestEnsureWatcherAccessToken_ProviderRefreshFails(t *testing.T) {
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_JGTM", "rt-revoked")

	store := newStoreWithTokens(t, &StoredTokens{
		AccessToken:    "old",
		RefreshToken:   "",
		OAuthExpiresAt: time.Now().Add(-1 * time.Hour),
	})
	prov := &stubProvider{oauthErr: errors.New("refresh_token revoked")}

	got, err := EnsureWatcherAccessToken(context.Background(), nil, store, prov, "JGtm")
	if err != nil {
		t.Errorf("erreur de refresh doit etre absorbee (mode degrade) — got err = %v", err)
	}
	if got != "" {
		t.Errorf("erreur de refresh doit retourner \"\" — got %q", got)
	}
}

// TestEnsureWatcherAccessToken_NilStore_NilProvider verifie les preconditions.
func TestEnsureWatcherAccessToken_NilArguments(t *testing.T) {
	prov := &stubProvider{}
	store := newStoreWithTokens(t, &StoredTokens{})

	if _, err := EnsureWatcherAccessToken(context.Background(), nil, nil, prov, "JGtm"); err == nil {
		t.Error("legacy store nil doit retourner une erreur")
	}
	if _, err := EnsureWatcherAccessToken(context.Background(), nil, store, nil, "JGtm"); err == nil {
		t.Error("provider nil doit retourner une erreur")
	}
}

// TestRefreshTokenFromEnv_NormalizationCases couvre les cas de normalisation
// les plus frequents : gamertag vide, ASCII pur, espaces, tirets, points,
// caracteres ASCII deja valides.
func TestRefreshTokenFromEnv_NormalizationCases(t *testing.T) {
	tests := []struct {
		gamertag string
		envKey   string
		value    string
		want     string
	}{
		{"JGtm", "SPNKR_OAUTH_REFRESH_TOKEN_JGTM", "rt-1", "rt-1"},
		{"My User", "SPNKR_OAUTH_REFRESH_TOKEN_MY_USER", "rt-2", "rt-2"},
		{"My-User", "SPNKR_OAUTH_REFRESH_TOKEN_MY_USER", "rt-3", "rt-3"},
		{"My.User", "SPNKR_OAUTH_REFRESH_TOKEN_MY_USER", "rt-4", "rt-4"},
		{"", "SPNKR_OAUTH_REFRESH_TOKEN_", "rt-5", ""}, // empty gamertag returns ""
	}

	for _, tt := range tests {
		t.Run(tt.gamertag, func(t *testing.T) {
			t.Setenv(tt.envKey, tt.value)
			got := RefreshTokenFromEnv(tt.gamertag)
			if got != tt.want {
				t.Errorf("RefreshTokenFromEnv(%q) = %q, want %q", tt.gamertag, got, tt.want)
			}
		})
	}
}

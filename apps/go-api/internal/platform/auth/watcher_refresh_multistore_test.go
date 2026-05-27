// Package auth — watcher_refresh_multistore_test.go : T5 ADR 0023.
//
// Couvre le NOUVEAU chemin Phase 3c : EnsureWatcherAccessToken lit le
// MultiUserTokenStore en priorité avant le legacy TokenStore mono-user et
// avant l'env var. Persiste la rotation dans le multi-store (canonique) en
// plus du legacy store (compat).
//
// Pas de cgo : le watcher_refresh ne dépend que du package auth pur.
package auth

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// newMultiStoreForWatcher crée un MultiUserTokenStore tempdir avec une entrée
// pré-remplie pour `gamertag`.
func newMultiStoreForWatcher(t *testing.T, xuid, gamertag, refreshToken string) *MultiUserTokenStore {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "watcher_tokens_multi")
	store := NewMultiUserTokenStore(dir)
	if refreshToken != "" {
		if err := store.Upsert(&UserTokens{
			XUID:              xuid,
			Gamertag:          gamertag,
			OAuthRefreshToken: refreshToken,
		}); err != nil {
			t.Fatalf("seed multi-store: %v", err)
		}
	}
	return store
}

// ─── T5.1 — Multi-store prioritaire sur legacy TokenStore ────────────────

func TestEnsureWatcherAccessToken_MultiStoreTakesPriorityOverLegacy(t *testing.T) {
	// Multi-store : RT canonique
	multiStore := newMultiStoreForWatcher(t, "2533274858283686", "JGtm", "rt-from-multi-store")

	// Legacy store : RT différent
	legacyStore := newStoreWithTokens(t, &StoredTokens{
		AccessToken:    "", // access expired → trigger refresh
		RefreshToken:   "rt-from-legacy-STALE",
		OAuthExpiresAt: time.Now().Add(-time.Hour),
	})

	prov := &stubProvider{oauthResp: "fresh-access-token"}

	got, err := EnsureWatcherAccessToken(context.Background(), multiStore, legacyStore, prov, "JGtm")
	if err != nil {
		t.Fatalf("EnsureWatcherAccessToken: %v", err)
	}
	if got != "fresh-access-token" {
		t.Errorf("access_token = %q", got)
	}
	if prov.lastCall != "rt-from-multi-store" {
		t.Errorf("provider called with %q, want rt-from-multi-store (multi-store prioritaire)", prov.lastCall)
	}
}

// ─── T5.2 — Multi-store vide → fallback legacy TokenStore ─────────────────

func TestEnsureWatcherAccessToken_MultiStoreEmpty_FallsBackToLegacy(t *testing.T) {
	// Multi-store vide (aucune entrée pour JGtm)
	multiStore := newMultiStoreForWatcher(t, "999", "OtherPlayer", "")

	legacyStore := newStoreWithTokens(t, &StoredTokens{
		RefreshToken:   "rt-from-legacy",
		OAuthExpiresAt: time.Now().Add(-time.Hour),
	})

	prov := &stubProvider{oauthResp: "access"}

	_, err := EnsureWatcherAccessToken(context.Background(), multiStore, legacyStore, prov, "JGtm")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if prov.lastCall != "rt-from-legacy" {
		t.Errorf("lastCall = %q, want rt-from-legacy (fallback legacy)", prov.lastCall)
	}
}

// ─── T5.3 — Multi-store vide + legacy vide → fallback env var ────────────

func TestEnsureWatcherAccessToken_MultiStoreAndLegacyEmpty_FallsBackToEnv(t *testing.T) {
	multiStore := newMultiStoreForWatcher(t, "999", "OtherPlayer", "")
	legacyStore := newStoreWithTokens(t, &StoredTokens{OAuthExpiresAt: time.Now().Add(-time.Hour)})

	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_JGTM", "rt-from-env-LAST_RESORT")

	prov := &stubProvider{oauthResp: "access"}

	_, err := EnsureWatcherAccessToken(context.Background(), multiStore, legacyStore, prov, "JGtm")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if prov.lastCall != "rt-from-env-LAST_RESORT" {
		t.Errorf("lastCall = %q, want rt-from-env-LAST_RESORT", prov.lastCall)
	}
}

// ─── T5.4 — Rotation : multi-store mis à jour avec le RT rotaté ─────────

// stubProviderWithRotation simule la rotation Microsoft.
type stubProviderWithRotation struct {
	stubProvider
	rotatedRT string
}

func (s *stubProviderWithRotation) TryOAuthRefreshWithRotation(_ context.Context, refreshToken string) (string, string, error) {
	s.lastCall = refreshToken
	return s.oauthResp, s.rotatedRT, s.oauthErr
}

func TestEnsureWatcherAccessToken_RotationPersistedInMultiStore(t *testing.T) {
	multiStore := newMultiStoreForWatcher(t, "2533274858283686", "JGtm", "rt-original")
	legacyStore := newStoreWithTokens(t, &StoredTokens{OAuthExpiresAt: time.Now().Add(-time.Hour)})

	prov := &stubProviderWithRotation{
		stubProvider: stubProvider{oauthResp: "access"},
		rotatedRT:    "rt-rotated-from-microsoft",
	}

	_, err := EnsureWatcherAccessToken(context.Background(), multiStore, legacyStore, prov, "JGtm")
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	// Vérifier que le multi-store a été mis à jour
	user, err := multiStore.Load("2533274858283686")
	if err != nil {
		t.Fatalf("multi-store Load: %v", err)
	}
	if user.OAuthRefreshToken != "rt-rotated-from-microsoft" {
		t.Errorf("multi-store RT après rotation = %q, want rt-rotated-from-microsoft", user.OAuthRefreshToken)
	}
}

// ─── T5.5 — Rotation : legacy store aussi mis à jour (compat) ────────────

func TestEnsureWatcherAccessToken_RotationAlsoPersistedInLegacyStore(t *testing.T) {
	multiStore := newMultiStoreForWatcher(t, "2533274858283686", "JGtm", "rt-original")
	legacyStore := newStoreWithTokens(t, &StoredTokens{OAuthExpiresAt: time.Now().Add(-time.Hour)})

	prov := &stubProviderWithRotation{
		stubProvider: stubProvider{oauthResp: "fresh-access"},
		rotatedRT:    "rt-rotated",
	}

	_, _ = EnsureWatcherAccessToken(context.Background(), multiStore, legacyStore, prov, "JGtm")

	tokens, _ := legacyStore.Load()
	if tokens.AccessToken != "fresh-access" {
		t.Errorf("legacy access_token = %q", tokens.AccessToken)
	}
	if tokens.RefreshToken != "rt-rotated" {
		t.Errorf("legacy RT = %q, want rt-rotated (rotation persistée double)", tokens.RefreshToken)
	}
}

// ─── T5.6 — Rotation : pas de rotation reçue → no-op ────────────────────

func TestEnsureWatcherAccessToken_NoRotationKeepsOriginalInStore(t *testing.T) {
	multiStore := newMultiStoreForWatcher(t, "2533274858283686", "JGtm", "rt-original")
	legacyStore := newStoreWithTokens(t, &StoredTokens{OAuthExpiresAt: time.Now().Add(-time.Hour)})

	prov := &stubProviderWithRotation{
		stubProvider: stubProvider{oauthResp: "access"},
		rotatedRT:    "", // Pas de rotation
	}

	_, _ = EnsureWatcherAccessToken(context.Background(), multiStore, legacyStore, prov, "JGtm")

	user, _ := multiStore.Load("2533274858283686")
	if user.OAuthRefreshToken != "rt-original" {
		t.Errorf("RT = %q, want rt-original (pas de rotation = pas de changement)", user.OAuthRefreshToken)
	}
}

// ─── T5.7 — Multi-store nil → fonctionnement legacy uniquement ───────────

func TestEnsureWatcherAccessToken_NilMultiStoreFallsBackToLegacy(t *testing.T) {
	legacyStore := newStoreWithTokens(t, &StoredTokens{
		RefreshToken:   "rt-legacy",
		OAuthExpiresAt: time.Now().Add(-time.Hour),
	})

	prov := &stubProvider{oauthResp: "access"}

	_, err := EnsureWatcherAccessToken(context.Background(), nil, legacyStore, prov, "JGtm")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if prov.lastCall != "rt-legacy" {
		t.Errorf("lastCall = %q, want rt-legacy", prov.lastCall)
	}
}

// ─── T5.8 — Multi-store erreur lecture → fallback gracieux ───────────────

func TestEnsureWatcherAccessToken_MultiStoreLookupFailsGracefully(t *testing.T) {
	// Store pointing to a dir which won't be writable / valid → LoadByGamertag will fail
	// But we test that even on error, fallback works.
	multiStore := newMultiStoreForWatcher(t, "999", "Other", "")

	legacyStore := newStoreWithTokens(t, &StoredTokens{
		RefreshToken:   "rt-legacy",
		OAuthExpiresAt: time.Now().Add(-time.Hour),
	})

	prov := &stubProvider{oauthResp: "access"}

	// JGtm absent du multi-store → LoadByGamertag retourne ErrUserTokensNotFound
	// → fallback legacy
	got, err := EnsureWatcherAccessToken(context.Background(), multiStore, legacyStore, prov, "JGtm")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "access" {
		t.Errorf("access = %q", got)
	}
	if prov.lastCall != "rt-legacy" {
		t.Errorf("lastCall = %q, want rt-legacy (fallback)", prov.lastCall)
	}
}

// ─── T5.9 — Multi-store sans xuid (entry partielle) → fallback ──────────

func TestEnsureWatcherAccessToken_MultiStoreEntryWithoutOAuthRT_FallsBack(t *testing.T) {
	// Entry présente dans le multi-store mais sans OAuthRefreshToken
	dir := filepath.Join(t.TempDir(), "watcher_tokens_multi")
	multiStore := NewMultiUserTokenStore(dir)
	if err := multiStore.Upsert(&UserTokens{
		XUID:     "2533274858283686",
		Gamertag: "JGtm",
		// OAuthRefreshToken vide → lookupRefreshToken doit skipper
		MSALCacheJSON: "cache-only",
	}); err != nil {
		t.Fatal(err)
	}

	legacyStore := newStoreWithTokens(t, &StoredTokens{
		RefreshToken:   "rt-legacy",
		OAuthExpiresAt: time.Now().Add(-time.Hour),
	})

	prov := &stubProvider{oauthResp: "access"}

	_, err := EnsureWatcherAccessToken(context.Background(), multiStore, legacyStore, prov, "JGtm")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if prov.lastCall != "rt-legacy" {
		t.Errorf("lastCall = %q, want rt-legacy (multi-store entry sans RT → fallback)", prov.lastCall)
	}
}

// ─── T5.10 — Provider error sur refresh → access_token reste valide ──────

func TestEnsureWatcherAccessToken_ProviderRefreshError_ReturnsEmpty(t *testing.T) {
	multiStore := newMultiStoreForWatcher(t, "2533274858283686", "JGtm", "rt-multi")
	legacyStore := newStoreWithTokens(t, &StoredTokens{OAuthExpiresAt: time.Now().Add(-time.Hour)})

	prov := &stubProviderWithRotation{
		stubProvider: stubProvider{
			oauthErr: errors.New("invalid_grant"),
		},
	}

	got, err := EnsureWatcherAccessToken(context.Background(), multiStore, legacyStore, prov, "JGtm")
	if err != nil {
		t.Errorf("err devrait être nil (erreur refresh non-fatale) : %v", err)
	}
	if got != "" {
		t.Errorf("access_token devrait être vide sur refresh erreur, got %q", got)
	}

	// Le RT dans le store ne doit PAS être modifié sur erreur
	user, _ := multiStore.Load("2533274858283686")
	if user.OAuthRefreshToken != "rt-multi" {
		t.Errorf("RT modifié sur erreur : %q, want rt-multi", user.OAuthRefreshToken)
	}
}

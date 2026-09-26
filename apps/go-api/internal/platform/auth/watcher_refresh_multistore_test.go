// Package auth — watcher_refresh_multistore_test.go : T5 ADR 0023.
//
// Couvre le chemin Phase 3c, durci en Phase 5 (2026-08-25) :
// EnsureWatcherAccessToken lit le refresh_token du MultiUserTokenStore — SEULE
// source — et y persiste la rotation. Le store mono-user (watcher_tokens.json)
// ne reçoit que l'access_token courant.
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

// ─── T5.1 — Le RT vient du MultiUserTokenStore ────────────────────────────

func TestEnsureWatcherAccessToken_UsesMultiStoreRefreshToken(t *testing.T) {
	multiStore := newMultiStoreForWatcher(t, "2533274858283686", "JGtm", "rt-from-multi-store")
	watcherState := newStoreWithTokens(t, &StoredTokens{
		AccessToken:    "", // access expiré → déclenche le refresh
		OAuthExpiresAt: time.Now().Add(-time.Hour),
	})

	prov := &stubProvider{oauthResp: "fresh-access-token"}

	got, err := EnsureWatcherAccessToken(context.Background(), multiStore, watcherState, prov, "JGtm")
	if err != nil {
		t.Fatalf("EnsureWatcherAccessToken: %v", err)
	}
	if got != "fresh-access-token" {
		t.Errorf("access_token = %q", got)
	}
	if prov.lastCall != "rt-from-multi-store" {
		t.Errorf("provider called with %q, want rt-from-multi-store", prov.lastCall)
	}
}

// ─── T5.2 — Store sans entrée pour ce gamertag → aucune source (Phase 5) ──

func TestEnsureWatcherAccessToken_MultiStoreEmpty_NoFallback(t *testing.T) {
	multiStore := newMultiStoreForWatcher(t, "999", "OtherPlayer", "")
	watcherState := newStoreWithTokens(t, &StoredTokens{OAuthExpiresAt: time.Now().Add(-time.Hour)})

	prov := &stubProvider{oauthResp: "access"}

	got, err := EnsureWatcherAccessToken(context.Background(), multiStore, watcherState, prov, "JGtm")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "" {
		t.Errorf("access_token = %q, want vide (aucune source legacy depuis Phase 5)", got)
	}
	if prov.lastCall != "" {
		t.Errorf("lastCall = %q — aucun RT ne doit être tenté", prov.lastCall)
	}
}

// ─── T5.3 — env var ignorée (source supprimée Phase 5) ────────────────────

func TestEnsureWatcherAccessToken_EnvVarIgnored(t *testing.T) {
	multiStore := newMultiStoreForWatcher(t, "999", "OtherPlayer", "")
	watcherState := newStoreWithTokens(t, &StoredTokens{OAuthExpiresAt: time.Now().Add(-time.Hour)})

	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_JGTM", "rt-from-env-DOIT-ETRE-IGNORE")

	prov := &stubProvider{oauthResp: "access"}

	got, err := EnsureWatcherAccessToken(context.Background(), multiStore, watcherState, prov, "JGtm")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "" || prov.lastCall != "" {
		t.Errorf("env var servie (got=%q lastCall=%q) — la source env est supprimée (ADR 0023 Phase 5)",
			got, prov.lastCall)
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
	watcherState := newStoreWithTokens(t, &StoredTokens{OAuthExpiresAt: time.Now().Add(-time.Hour)})

	prov := &stubProviderWithRotation{
		stubProvider: stubProvider{oauthResp: "access"},
		rotatedRT:    "rt-rotated-from-microsoft",
	}

	_, err := EnsureWatcherAccessToken(context.Background(), multiStore, watcherState, prov, "JGtm")
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	user, err := multiStore.Load("2533274858283686")
	if err != nil {
		t.Fatalf("multi-store Load: %v", err)
	}
	if user.OAuthRefreshToken != "rt-rotated-from-microsoft" {
		t.Errorf("multi-store RT après rotation = %q, want rt-rotated-from-microsoft", user.OAuthRefreshToken)
	}
}

// ─── T5.5 — L'état watcher ne reçoit QUE l'access_token ────────────────────

func TestEnsureWatcherAccessToken_WatcherStateGetsAccessTokenOnly(t *testing.T) {
	multiStore := newMultiStoreForWatcher(t, "2533274858283686", "JGtm", "rt-original")
	watcherState := newStoreWithTokens(t, &StoredTokens{OAuthExpiresAt: time.Now().Add(-time.Hour)})

	prov := &stubProviderWithRotation{
		stubProvider: stubProvider{oauthResp: "fresh-access"},
		rotatedRT:    "rt-rotated",
	}

	_, _ = EnsureWatcherAccessToken(context.Background(), multiStore, watcherState, prov, "JGtm")

	tokens, _ := watcherState.Load()
	if tokens.AccessToken != "fresh-access" {
		t.Errorf("access_token du watcher = %q, want fresh-access", tokens.AccessToken)
	}
	if !tokens.IsOAuthValid(time.Minute) {
		t.Error("OAuthExpiresAt du watcher devrait être rafraîchi")
	}
}

// ─── T5.6 — Rotation : pas de rotation reçue → no-op ────────────────────

func TestEnsureWatcherAccessToken_NoRotationKeepsOriginalInStore(t *testing.T) {
	multiStore := newMultiStoreForWatcher(t, "2533274858283686", "JGtm", "rt-original")
	watcherState := newStoreWithTokens(t, &StoredTokens{OAuthExpiresAt: time.Now().Add(-time.Hour)})

	prov := &stubProviderWithRotation{
		stubProvider: stubProvider{oauthResp: "access"},
		rotatedRT:    "", // Pas de rotation
	}

	_, _ = EnsureWatcherAccessToken(context.Background(), multiStore, watcherState, prov, "JGtm")

	user, _ := multiStore.Load("2533274858283686")
	if user.OAuthRefreshToken != "rt-original" {
		t.Errorf("RT = %q, want rt-original (pas de rotation = pas de changement)", user.OAuthRefreshToken)
	}
}

// ─── T5.7 — Multi-store nil → aucune source ─────────────────────────────

func TestEnsureWatcherAccessToken_NilMultiStoreNoSource(t *testing.T) {
	watcherState := newStoreWithTokens(t, &StoredTokens{OAuthExpiresAt: time.Now().Add(-time.Hour)})

	prov := &stubProvider{oauthResp: "access"}

	got, err := EnsureWatcherAccessToken(context.Background(), nil, watcherState, prov, "JGtm")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "" || prov.lastCall != "" {
		t.Errorf("sans multi-store il n'y a plus de source (got=%q lastCall=%q)", got, prov.lastCall)
	}
}

// ─── T5.8 — Entrée store sans OAuthRefreshToken → aucune source ─────────

func TestEnsureWatcherAccessToken_MultiStoreEntryWithoutOAuthRT(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "watcher_tokens_multi")
	multiStore := NewMultiUserTokenStore(dir)
	if err := multiStore.Upsert(&UserTokens{
		XUID:     "2533274858283686",
		Gamertag: "JGtm",
		// OAuthRefreshToken vide → lookupRefreshToken doit skipper
		XSTSToken: "xsts-only",
	}); err != nil {
		t.Fatal(err)
	}

	watcherState := newStoreWithTokens(t, &StoredTokens{OAuthExpiresAt: time.Now().Add(-time.Hour)})
	prov := &stubProvider{oauthResp: "access"}

	got, err := EnsureWatcherAccessToken(context.Background(), multiStore, watcherState, prov, "JGtm")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "" || prov.lastCall != "" {
		t.Errorf("entrée sans RT → aucune source (got=%q lastCall=%q)", got, prov.lastCall)
	}
}

// ─── T5.9 — Provider error sur refresh → RT du store intact ──────────────

func TestEnsureWatcherAccessToken_ProviderRefreshError_ReturnsEmpty(t *testing.T) {
	multiStore := newMultiStoreForWatcher(t, "2533274858283686", "JGtm", "rt-multi")
	watcherState := newStoreWithTokens(t, &StoredTokens{OAuthExpiresAt: time.Now().Add(-time.Hour)})

	prov := &stubProviderWithRotation{
		stubProvider: stubProvider{
			oauthErr: errors.New("invalid_grant"),
		},
	}

	got, err := EnsureWatcherAccessToken(context.Background(), multiStore, watcherState, prov, "JGtm")
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

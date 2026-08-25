// Package auth — access_token_store_first_test.go : tests de la résolution
// d'access_token depuis le MultiUserTokenStore (source unique ADR 0023).
//
// ADR 0023 Phase 5 (2026-08-25) : les cas « legacy adopté / télémétrie
// legacy_source_used » ont disparu avec les sources legacy elles-mêmes. Ce qui
// reste couvert : le store résout, la rotation est persistée, l'erreur OAuth est
// surfacée, l'absence de source est un skip non fatal, et un store illisible est
// LOGUÉ (jamais avalé).
package auth

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

// seedStoreOAuth crée une entrée store avec un OAuth refresh token.
func seedStoreOAuth(t *testing.T, store *MultiUserTokenStore, xuid, gamertag, rt string) {
	t.Helper()
	if err := store.Upsert(&UserTokens{XUID: xuid, Gamertag: gamertag, OAuthRefreshToken: rt}); err != nil {
		t.Fatal(err)
	}
}

// TestResolveMSAccessTokenStoreFirst_StoreResolves : le RT du store est le SEUL
// utilisé (non-régression de l'incident 2026-07-12, où un résidu sync_meta était
// servi à la place du store pour les 4 joueurs prod).
func TestResolveMSAccessTokenStoreFirst_StoreResolves(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	seedStoreOAuth(t, store, "111", "Alice", "rt-store-FRESH")
	prov := &fakeProvider{oauthAccess: "at-store"}

	at, err := ResolveMSAccessTokenStoreFirst(context.Background(), prov, store, "111", "Alice")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if at != "at-store" {
		t.Errorf("access_token = %q, want at-store (résolu par le store)", at)
	}
	if prov.lastOAuthRT != "rt-store-FRESH" {
		t.Errorf("RT utilisé = %q, want rt-store-FRESH", prov.lastOAuthRT)
	}
}

// TestResolveMSAccessTokenStoreFirst_StoreEmpty_NoFallback : sans entrée store,
// plus AUCUNE source de repli (Phase 5) → skip non fatal, aucun appel provider.
func TestResolveMSAccessTokenStoreFirst_StoreEmpty_NoFallback(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	prov := &fakeProvider{oauthAccess: "at-jamais-servi"}

	at, err := ResolveMSAccessTokenStoreFirst(context.Background(), prov, store, "111", "Alice")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if at != "" {
		t.Errorf("access_token = %q, want vide (aucune source legacy)", at)
	}
	if prov.lastOAuthRT != "" {
		t.Errorf("provider appelé avec %q — aucun RT ne doit être tenté", prov.lastOAuthRT)
	}
}

// TestResolveMSAccessTokenStoreFirst_StoreRotationPersisted : le store résout via
// OAuth et Microsoft rotate le RT → le RT rotaté est persisté dans le store.
func TestResolveMSAccessTokenStoreFirst_StoreRotationPersisted(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	seedStoreOAuth(t, store, "111", "Alice", "rt-v1")
	prov := &fakeProvider{oauthAccess: "at-store", oauthRotated: "rt-v2-rotated"}

	if _, err := ResolveMSAccessTokenStoreFirst(context.Background(), prov, store, "111", "Alice"); err != nil {
		t.Fatalf("err = %v", err)
	}
	user, err := store.Load("111")
	if err != nil || user == nil {
		t.Fatalf("Load: %v", err)
	}
	if user.OAuthRefreshToken != "rt-v2-rotated" {
		t.Errorf("RT store après rotation = %q, want rt-v2-rotated", user.OAuthRefreshToken)
	}
}

// TestResolveMSAccessTokenStoreFirst_OAuthErrorSurfaced : une erreur OAuth
// sous-jacente (ex. invalid_grant) est enveloppée dans l'erreur retournée pour le
// diagnostic — pas un skip opaque.
func TestResolveMSAccessTokenStoreFirst_OAuthErrorSurfaced(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	seedStoreOAuth(t, store, "111", "Alice", "rt-dead")
	sentinel := errors.New("invalid_grant")
	prov := &fakeProvider{oauthErr: sentinel}

	at, err := ResolveMSAccessTokenStoreFirst(context.Background(), prov, store, "111", "Alice")
	if at != "" {
		t.Errorf("access_token = %q, want vide", at)
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, doit envelopper invalid_grant", err)
	}
}

// TestResolveMSAccessTokenStoreFirst_NoSourceReturnsNilNil : aucune source
// exploitable, aucune erreur → ("", nil) (skip légitime, non fatal).
func TestResolveMSAccessTokenStoreFirst_NoSourceReturnsNilNil(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	prov := &fakeProvider{}
	at, err := ResolveMSAccessTokenStoreFirst(context.Background(), prov, store, "111", "Alice")
	if at != "" || err != nil {
		t.Errorf("(at, err) = (%q, %v), want (\"\", nil)", at, err)
	}
}

// TestResolveMSAccessTokenStoreFirst_ProviderNil : garde défensive.
func TestResolveMSAccessTokenStoreFirst_ProviderNil(t *testing.T) {
	_, err := ResolveMSAccessTokenStoreFirst(context.Background(), nil, nil, "111", "Alice")
	if err == nil {
		t.Error("err = nil, want provider nil")
	}
}

// TestResolveMSAccessTokenStoreFirst_StoreLoadError_Logged — AU3 (revue 2026-07) :
// un échec de lecture du store canonique ne doit jamais être AVALÉ. Depuis la
// Phase 5, un store illisible signifie plus aucune auth possible pour ce joueur :
// le log ERROR est la seule trace du diagnostic.
func TestResolveMSAccessTokenStoreFirst_StoreLoadError_Logged(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)

	store := NewMultiUserTokenStore(tempTokenDir(t))
	prov := &fakeProvider{}

	// xuid unsafe → store.Load renvoie une erreur ("xuid invalide") → retour
	// ("", nil) (skip légitime), mais l'échec store DOIT être logué.
	if _, err := ResolveMSAccessTokenStoreFirst(context.Background(), prov, store, "../escape", "Alice"); err != nil {
		t.Fatalf("attendu skip (\"\", nil), got err=%v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "échec lecture store canonique") {
		t.Errorf("attendu un log ERROR sur l'échec store.Load (AU3), got:\n%s", out)
	}
	if !strings.Contains(out, "\"level\":\"ERROR\"") {
		t.Errorf("le log de l'échec store doit être de niveau ERROR (AU3), got:\n%s", out)
	}
}

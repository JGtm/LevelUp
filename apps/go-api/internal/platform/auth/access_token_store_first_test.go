// Package auth — access_token_store_first_test.go : tests de l'ordre de
// résolution store-first de ResolveMSAccessTokenStoreFirst et de la propriété
// clé pour le gate D2 (ADR 0023) : la télémétrie legacy_source_used ne se
// déclenche QU'EN vraie absence de token store — jamais quand le store couvre le
// joueur (reproduction du scénario des 4 joueurs prod, incident 2026-07-12).
package auth

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"levelup/go-api/internal/observability"
)

// seedStoreOAuth crée une entrée store avec un OAuth refresh token (pas de MSAL).
func seedStoreOAuth(t *testing.T, store *MultiUserTokenStore, xuid, gamertag, rt string) {
	t.Helper()
	if err := store.Upsert(&UserTokens{XUID: xuid, Gamertag: gamertag, OAuthRefreshToken: rt}); err != nil {
		t.Fatal(err)
	}
}

// TestResolveMSAccessTokenStoreFirst_StoreCovers_NoLegacyTelemetry est le test de
// non-régression de l'incident prod : le store couvre le joueur, un résidu legacy
// existe AUSSI (sync_meta) mais il ne doit JAMAIS être servi ni compté. C'est
// exactement la situation des 4 joueurs (JGtm, Madina97294, Chocoboflor,
// XxDaemonGamerxX) : watcher_tokens/{xuid}.json valide + sync_meta.oauth_refresh_token
// résiduel. Attendu : token du store, compteur duckdb_oauth INCHANGÉ.
func TestResolveMSAccessTokenStoreFirst_StoreCovers_NoLegacyTelemetry(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	seedStoreOAuth(t, store, "111", "Alice", "rt-store-FRESH")
	prov := &fakeProvider{oauthAccess: "at-store"}

	before := observability.LoadCounter("legacy_source_used_" + observability.LegacySourceDuckDBOAuth)
	beforeMSAL := observability.LoadCounter("legacy_source_used_" + observability.LegacySourceDuckDBMSAL)

	at, err := ResolveMSAccessTokenStoreFirst(context.Background(), prov, store, "111", "Alice",
		LegacyAuthInputs{OAuthRT: "rt-legacy-STALE", MSALCache: "msal-legacy", Source: "player_db.sync_meta"})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if at != "at-store" {
		t.Errorf("access_token = %q, want at-store (résolu par le store)", at)
	}
	if prov.lastOAuthRT != "rt-store-FRESH" {
		t.Errorf("RT utilisé = %q, want rt-store-FRESH (le résidu legacy ne doit pas servir)", prov.lastOAuthRT)
	}
	if after := observability.LoadCounter("legacy_source_used_" + observability.LegacySourceDuckDBOAuth); after != before {
		t.Errorf("compteur duckdb_oauth = %d, attendu %d (store couvre → aucun comptage legacy)", after, before)
	}
	if after := observability.LoadCounter("legacy_source_used_" + observability.LegacySourceDuckDBMSAL); after != beforeMSAL {
		t.Errorf("compteur duckdb_msal = %d, attendu %d (store couvre → aucun comptage legacy)", after, beforeMSAL)
	}
}

// TestResolveMSAccessTokenStoreFirst_StoreEmpty_LegacyServesAndCounts : sans
// entrée store, le résidu legacy sync_meta est réellement adopté → token servi +
// compteur duckdb_oauth +1 (cas dégradé légitime, seul cas où la télémétrie doit
// bouger).
func TestResolveMSAccessTokenStoreFirst_StoreEmpty_LegacyServesAndCounts(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	prov := &fakeProvider{oauthAccess: "at-legacy"}

	before := observability.LoadCounter("legacy_source_used_" + observability.LegacySourceDuckDBOAuth)

	at, err := ResolveMSAccessTokenStoreFirst(context.Background(), prov, store, "111", "Alice",
		LegacyAuthInputs{OAuthRT: "rt-legacy", Source: "player_db.sync_meta"})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if at != "at-legacy" {
		t.Errorf("access_token = %q, want at-legacy", at)
	}
	if after := observability.LoadCounter("legacy_source_used_" + observability.LegacySourceDuckDBOAuth); after != before+1 {
		t.Errorf("compteur duckdb_oauth = %d, attendu %d (adoption legacy → +1)", after, before+1)
	}
}

// TestResolveMSAccessTokenStoreFirst_EnvRTUsesEnvCounter : un RT legacy marqué
// OAuthRTFromEnv est compté sous env_oauth, pas duckdb_oauth.
func TestResolveMSAccessTokenStoreFirst_EnvRTUsesEnvCounter(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	prov := &fakeProvider{oauthAccess: "at-env"}

	beforeEnv := observability.LoadCounter("legacy_source_used_" + observability.LegacySourceEnvOAuth)
	beforeDuck := observability.LoadCounter("legacy_source_used_" + observability.LegacySourceDuckDBOAuth)

	_, err := ResolveMSAccessTokenStoreFirst(context.Background(), prov, store, "111", "Alice",
		LegacyAuthInputs{OAuthRT: "rt-env", OAuthRTFromEnv: true, Source: "env_var"})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if after := observability.LoadCounter("legacy_source_used_" + observability.LegacySourceEnvOAuth); after != beforeEnv+1 {
		t.Errorf("compteur env_oauth = %d, attendu %d", after, beforeEnv+1)
	}
	if after := observability.LoadCounter("legacy_source_used_" + observability.LegacySourceDuckDBOAuth); after != beforeDuck {
		t.Errorf("compteur duckdb_oauth = %d, attendu %d (RT env ne compte pas duckdb)", after, beforeDuck)
	}
}

// TestResolveMSAccessTokenStoreFirst_StoreRotationPersisted : le store résout via
// OAuth et Microsoft rotate le RT → le RT rotaté est persisté dans le store.
func TestResolveMSAccessTokenStoreFirst_StoreRotationPersisted(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	seedStoreOAuth(t, store, "111", "Alice", "rt-v1")
	prov := &fakeProvider{oauthAccess: "at-store", oauthRotated: "rt-v2-rotated"}

	if _, err := ResolveMSAccessTokenStoreFirst(context.Background(), prov, store, "111", "Alice",
		LegacyAuthInputs{}); err != nil {
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

// TestResolveMSAccessTokenStoreFirst_LegacyRotationMigratesToStore : le RT legacy
// est adopté et Microsoft rotate → le RT rotaté est écrit dans le store (migration
// vers le canonique : au refresh suivant, le store résout, la télémétrie s'éteint).
func TestResolveMSAccessTokenStoreFirst_LegacyRotationMigratesToStore(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	prov := &fakeProvider{oauthAccess: "at-legacy", oauthRotated: "rt-migrated"}

	if _, err := ResolveMSAccessTokenStoreFirst(context.Background(), prov, store, "111", "Alice",
		LegacyAuthInputs{OAuthRT: "rt-legacy", Source: "player_db.sync_meta"}); err != nil {
		t.Fatalf("err = %v", err)
	}
	user, err := store.Load("111")
	if err != nil || user == nil {
		t.Fatalf("Load après migration: %v", err)
	}
	if user.OAuthRefreshToken != "rt-migrated" {
		t.Errorf("RT store après migration legacy = %q, want rt-migrated", user.OAuthRefreshToken)
	}
}

// TestResolveMSAccessTokenStoreFirst_OAuthErrorSurfaced : une erreur OAuth
// sous-jacente (ex. invalid_grant) est enveloppée dans l'erreur retournée pour le
// diagnostic — pas un skip opaque.
func TestResolveMSAccessTokenStoreFirst_OAuthErrorSurfaced(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	sentinel := errors.New("invalid_grant")
	prov := &fakeProvider{oauthErr: sentinel}

	at, err := ResolveMSAccessTokenStoreFirst(context.Background(), prov, store, "111", "Alice",
		LegacyAuthInputs{OAuthRT: "rt-dead", Source: "player_db.sync_meta"})
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
	at, err := ResolveMSAccessTokenStoreFirst(context.Background(), prov, store, "111", "Alice", LegacyAuthInputs{})
	if at != "" || err != nil {
		t.Errorf("(at, err) = (%q, %v), want (\"\", nil)", at, err)
	}
}

// TestResolveMSAccessTokenStoreFirst_ProviderNil : garde défensive.
func TestResolveMSAccessTokenStoreFirst_ProviderNil(t *testing.T) {
	_, err := ResolveMSAccessTokenStoreFirst(context.Background(), nil, nil, "111", "Alice", LegacyAuthInputs{})
	if err == nil {
		t.Error("err = nil, want provider nil")
	}
}

// TestResolveMSAccessTokenStoreFirst_StoreLoadError_Logged — AU3 (revue 2026-07) :
// un échec de lecture du store canonique ne doit plus être AVALÉ. Sans ce log, une
// bascule legacy déclenchée par un store illisible/corrompu était invisible → la
// télémétrie legacy_source_used (gate D2) devenait trompeuse.
func TestResolveMSAccessTokenStoreFirst_StoreLoadError_Logged(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)

	store := NewMultiUserTokenStore(tempTokenDir(t))
	prov := &fakeProvider{}

	// xuid unsafe → store.Load renvoie une erreur ("xuid invalide"). Aucune source
	// legacy → retour ("", nil) (skip légitime), mais l'échec store DOIT être logué.
	if _, err := ResolveMSAccessTokenStoreFirst(context.Background(), prov, store, "../escape", "Alice", LegacyAuthInputs{}); err != nil {
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

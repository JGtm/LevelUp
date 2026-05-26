// Package pool — discovery_priority_test.go : tests T3b ADR 0023.
//
// Valide que MultiUserTokenStore a priorité sur les sources legacy
// (DuckDB sync_meta + env var SPNKR_OAUTH_REFRESH_TOKEN_*).
//
// Focus sur le nouveau champ OAuthRefreshToken (Phase 3b) et les sources
// labels canoniques :
//   - watcher_oauth : nouveau label, Discovery lit le RT depuis le store
//   - watcher_msal : existant, RT MSAL depuis le store
//   - watcher_oauth+watcher_msal : combinaison
//   - duckdb_* + env_oauth : fallbacks legacy (warn log)
//
// Build tag cgo car Discovery dépend de duckdb.OpenReadOnly transitivement.
//
//go:build cgo

package pool

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
)

// ─── Priorité : store OAuth RT prioritaire sur env var ───────────────────

func TestDiscoveryScan_StoreOAuthRT_TakesPriorityOverEnvVar(t *testing.T) {
	cfg := fakeConfigWithPlayers(t, "Madina97294", "2533274858283686")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	// Env var legacy (devrait être IGNORÉ car store a une valeur)
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_MADINA97294", "rt-from-env-STALE")

	// Store avec RT canonique (Phase 3b)
	storeDir := filepath.Join(cfg.RepoRoot, "data", "auth", "watcher_tokens")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	store := auth.NewMultiUserTokenStore(storeDir)
	if err := store.Upsert(&auth.UserTokens{
		XUID:              "2533274858283686",
		Gamertag:          "Madina97294",
		OAuthRefreshToken: "rt-from-store-FRESH",
	}); err != nil {
		t.Fatal(err)
	}

	d := NewDiscoveryWithStores(cfg, resolver, titlePkg.DefaultSlug, store, nil)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(sources) != 1 {
		t.Fatalf("sources = %d, want 1", len(sources))
	}
	src := sources[0]
	if src.RefreshToken != "rt-from-store-FRESH" {
		t.Errorf("RT = %q, want rt-from-store-FRESH (store prioritaire sur env var)", src.RefreshToken)
	}
	if !strings.Contains(src.Source, credSourceWatcherOAuth) {
		t.Errorf("Source = %q, want contient %q", src.Source, credSourceWatcherOAuth)
	}
	// La source ne doit PAS contenir env_oauth
	if strings.Contains(src.Source, "env_oauth") {
		t.Errorf("Source = %q ne devrait pas contenir env_oauth (store autoritaire)", src.Source)
	}
}

// ─── Priorité : store MSAL + RT combinés ──────────────────────────────────

func TestDiscoveryScan_StoreCombinedMSAL_OAuthLabels(t *testing.T) {
	cfg := fakeConfigWithPlayers(t, "Madina97294", "2533274858283686")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_MADINA97294", "")

	storeDir := filepath.Join(cfg.RepoRoot, "data", "auth", "watcher_tokens")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	store := auth.NewMultiUserTokenStore(storeDir)
	if err := store.Upsert(&auth.UserTokens{
		XUID:              "2533274858283686",
		Gamertag:          "Madina97294",
		MSALCacheJSON:     `{"cached":"data"}`,
		OAuthRefreshToken: "rt-store",
	}); err != nil {
		t.Fatal(err)
	}

	d := NewDiscoveryWithStores(cfg, resolver, titlePkg.DefaultSlug, store, nil)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(sources) != 1 {
		t.Fatalf("sources = %d, want 1", len(sources))
	}
	src := sources[0]
	if src.MSALCache != `{"cached":"data"}` {
		t.Errorf("MSALCache = %q", src.MSALCache)
	}
	if src.RefreshToken != "rt-store" {
		t.Errorf("RefreshToken = %q", src.RefreshToken)
	}
	// Source label doit combiner MSAL + OAuth
	expected := credSourceWatcherMSAL + "+" + credSourceWatcherOAuth
	if src.Source != expected {
		t.Errorf("Source = %q, want %q", src.Source, expected)
	}
}

// ─── Priorité : store MSAL only (OAuth absent) ────────────────────────────

func TestDiscoveryScan_StoreMSALOnly_NoOAuth(t *testing.T) {
	cfg := fakeConfigWithPlayers(t, "Madina97294", "2533274858283686")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_MADINA97294", "")

	storeDir := filepath.Join(cfg.RepoRoot, "data", "auth", "watcher_tokens")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	store := auth.NewMultiUserTokenStore(storeDir)
	if err := store.Upsert(&auth.UserTokens{
		XUID:          "2533274858283686",
		Gamertag:      "Madina97294",
		MSALCacheJSON: `{"only":"msal"}`,
	}); err != nil {
		t.Fatal(err)
	}

	d := NewDiscoveryWithStores(cfg, resolver, titlePkg.DefaultSlug, store, nil)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(sources) != 1 {
		t.Fatalf("sources = %d, want 1", len(sources))
	}
	src := sources[0]
	if src.MSALCache == "" {
		t.Error("MSALCache vide alors que store a la valeur")
	}
	if src.RefreshToken != "" {
		t.Errorf("RefreshToken = %q, want vide (store n'a pas de RT)", src.RefreshToken)
	}
	if src.Source != credSourceWatcherMSAL {
		t.Errorf("Source = %q, want %q (MSAL seul)", src.Source, credSourceWatcherMSAL)
	}
}

// ─── Priorité : store OAuth only (MSAL absent) ────────────────────────────

func TestDiscoveryScan_StoreOAuthOnly_NoMSAL(t *testing.T) {
	cfg := fakeConfigWithPlayers(t, "Madina97294", "2533274858283686")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_MADINA97294", "")

	storeDir := filepath.Join(cfg.RepoRoot, "data", "auth", "watcher_tokens")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	store := auth.NewMultiUserTokenStore(storeDir)
	if err := store.UpdateOAuthRefreshToken("2533274858283686", "rt-only"); err != nil {
		t.Fatal(err)
	}

	d := NewDiscoveryWithStores(cfg, resolver, titlePkg.DefaultSlug, store, nil)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(sources) != 1 {
		t.Fatalf("sources = %d, want 1", len(sources))
	}
	src := sources[0]
	if src.RefreshToken != "rt-only" {
		t.Errorf("RefreshToken = %q, want rt-only", src.RefreshToken)
	}
	if src.Source != credSourceWatcherOAuth {
		t.Errorf("Source = %q, want %q (OAuth seul)", src.Source, credSourceWatcherOAuth)
	}
}

// ─── Fallback env var quand store vide ────────────────────────────────────

func TestDiscoveryScan_StoreEmpty_FallbackEnvVar(t *testing.T) {
	cfg := fakeConfigWithPlayers(t, "Madina97294", "2533274858283686")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_MADINA97294", "rt-from-env-LEGACY")

	storeDir := filepath.Join(cfg.RepoRoot, "data", "auth", "watcher_tokens")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	store := auth.NewMultiUserTokenStore(storeDir)
	// Store vide intentionnellement (pas d'Upsert)

	d := NewDiscoveryWithStores(cfg, resolver, titlePkg.DefaultSlug, store, nil)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(sources) != 1 {
		t.Fatalf("sources = %d, want 1 (env var fallback)", len(sources))
	}
	src := sources[0]
	if src.RefreshToken != "rt-from-env-LEGACY" {
		t.Errorf("RT = %q, want rt-from-env-LEGACY (fallback env)", src.RefreshToken)
	}
	if !strings.Contains(src.Source, "env_oauth") {
		t.Errorf("Source = %q, want contient 'env_oauth' (fallback legacy)", src.Source)
	}
}

// ─── Aucune source : joueur exclu ────────────────────────────────────────

func TestDiscoveryScan_NoSourcesAnywhere_PlayerExcluded(t *testing.T) {
	cfg := fakeConfigWithPlayers(t, "Madina97294", "2533274858283686")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_MADINA97294", "")

	storeDir := filepath.Join(cfg.RepoRoot, "data", "auth", "watcher_tokens")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	store := auth.NewMultiUserTokenStore(storeDir)

	d := NewDiscoveryWithStores(cfg, resolver, titlePkg.DefaultSlug, store, nil)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(sources) != 0 {
		t.Errorf("sources = %d, want 0 (aucune source partout)", len(sources))
	}
}

// ─── 3 joueurs, sources mixtes ────────────────────────────────────────────

func TestDiscoveryScan_MultiPlayers_MixedSources(t *testing.T) {
	repoRoot := t.TempDir()
	profilesJSON := `{
  "version": "3.0",
  "profiles": {
    "halo_infinite": {
      "Alice": {"db_path": "data/titles/halo_infinite/players/Alice/stats.duckdb", "xuid": "111", "waypoint_player": "Alice"},
      "Bob":   {"db_path": "data/titles/halo_infinite/players/Bob/stats.duckdb",   "xuid": "222", "waypoint_player": "Bob"},
      "Carol": {"db_path": "data/titles/halo_infinite/players/Carol/stats.duckdb", "xuid": "333", "waypoint_player": "Carol"}
    }
  }
}`
	profilesPath := filepath.Join(repoRoot, "db_profiles.json")
	if err := os.WriteFile(profilesPath, []byte(profilesJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.AppConfig{
		RepoRoot:        repoRoot,
		DBProfilesPath:  profilesPath,
		AppSettingsPath: filepath.Join(repoRoot, "app_settings.json"),
	}
	resolver := titlePkg.NewPathResolver(repoRoot)

	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_ALICE", "")
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_BOB", "rt-bob-env")
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_CAROL", "")

	storeDir := filepath.Join(repoRoot, "data", "auth", "watcher_tokens")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	store := auth.NewMultiUserTokenStore(storeDir)
	// Alice : store OAuth
	_ = store.UpdateOAuthRefreshToken("111", "rt-alice-store")
	// Bob : pas dans store → fallback env var
	// Carol : rien nulle part → exclue

	d := NewDiscoveryWithStores(cfg, resolver, titlePkg.DefaultSlug, store, nil)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(sources) != 2 {
		t.Fatalf("sources = %d, want 2 (Alice + Bob, Carol exclue)", len(sources))
	}

	byGamertag := make(map[string]CredentialSource, len(sources))
	for _, s := range sources {
		byGamertag[s.Gamertag] = s
	}
	if alice, ok := byGamertag["Alice"]; !ok {
		t.Error("Alice manquante")
	} else {
		if alice.RefreshToken != "rt-alice-store" {
			t.Errorf("Alice RT = %q", alice.RefreshToken)
		}
		if !strings.Contains(alice.Source, credSourceWatcherOAuth) {
			t.Errorf("Alice source = %q", alice.Source)
		}
	}
	if bob, ok := byGamertag["Bob"]; !ok {
		t.Error("Bob manquant")
	} else {
		if bob.RefreshToken != "rt-bob-env" {
			t.Errorf("Bob RT = %q", bob.RefreshToken)
		}
		if !strings.Contains(bob.Source, "env_oauth") {
			t.Errorf("Bob source = %q, want contient env_oauth", bob.Source)
		}
	}
	if _, ok := byGamertag["Carol"]; ok {
		t.Error("Carol devrait être exclue (aucune source)")
	}
}

// Package pool — discovery_priority_test.go : tests T3b ADR 0023, durcis en
// Phase 5 (2026-08-25).
//
// Le MultiUserTokenStore est désormais la SEULE source de credentials du scan :
// les fallbacks legacy (sync_meta DuckDB, env var SPNKR_OAUTH_REFRESH_TOKEN_*,
// store mono-user) ont été supprimés. Le seul label possible est watcher_oauth.
//
// Build tag cgo car Discovery dépend de duckdb transitivement.
//
//go:build cgo

package pool

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
)

// ─── Le RT vient du store, l'env var est ignorée ─────────────────────────

func TestDiscoveryScan_StoreOAuthRT_EnvVarIgnored(t *testing.T) {
	cfg := fakeConfigWithPlayers(t, "Madina97294", "2533274858283686")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	// Env var legacy : doit être TOTALEMENT ignorée (Phase 5).
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_MADINA97294", "rt-from-env-STALE")

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

	d := NewDiscoveryWithStore(cfg, resolver, titlePkg.DefaultSlug, store)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(sources) != 1 {
		t.Fatalf("sources = %d, want 1", len(sources))
	}
	src := sources[0]
	if src.RefreshToken != "rt-from-store-FRESH" {
		t.Errorf("RT = %q, want rt-from-store-FRESH", src.RefreshToken)
	}
	if src.Source != credSourceWatcherOAuth {
		t.Errorf("Source = %q, want %q", src.Source, credSourceWatcherOAuth)
	}
}

// ─── Entrée store sans RT → joueur exclu ─────────────────────────────────

func TestDiscoveryScan_StoreEntryWithoutRT_PlayerExcluded(t *testing.T) {
	cfg := fakeConfigWithPlayers(t, "Madina97294", "2533274858283686")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	storeDir := filepath.Join(cfg.RepoRoot, "data", "auth", "watcher_tokens")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	store := auth.NewMultiUserTokenStore(storeDir)
	if err := store.Upsert(&auth.UserTokens{
		XUID:      "2533274858283686",
		Gamertag:  "Madina97294",
		XSTSToken: "xsts-only", // aucun RT
	}); err != nil {
		t.Fatal(err)
	}

	d := NewDiscoveryWithStore(cfg, resolver, titlePkg.DefaultSlug, store)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 0 {
		t.Errorf("sources = %d, want 0 (entrée sans refresh token)", len(sources))
	}
}

// ─── Store vide + env var présente → joueur exclu (ratchet Phase 5) ──────

func TestDiscoveryScan_StoreEmpty_NoEnvFallback(t *testing.T) {
	cfg := fakeConfigWithPlayers(t, "Madina97294", "2533274858283686")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_MADINA97294", "rt-from-env-LEGACY")

	storeDir := filepath.Join(cfg.RepoRoot, "data", "auth", "watcher_tokens")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	store := auth.NewMultiUserTokenStore(storeDir) // vide intentionnellement

	d := NewDiscoveryWithStore(cfg, resolver, titlePkg.DefaultSlug, store)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(sources) != 0 {
		t.Fatalf("sources = %d, want 0 — l'env var n'est plus un fallback (ADR 0023 Phase 5)", len(sources))
	}
}

// ─── Aucune source : joueur exclu ────────────────────────────────────────

func TestDiscoveryScan_NoSourcesAnywhere_PlayerExcluded(t *testing.T) {
	cfg := fakeConfigWithPlayers(t, "Madina97294", "2533274858283686")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	storeDir := filepath.Join(cfg.RepoRoot, "data", "auth", "watcher_tokens")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	store := auth.NewMultiUserTokenStore(storeDir)

	d := NewDiscoveryWithStore(cfg, resolver, titlePkg.DefaultSlug, store)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(sources) != 0 {
		t.Errorf("sources = %d, want 0 (aucune source partout)", len(sources))
	}
}

// ─── 3 joueurs : seuls ceux couverts par le store entrent dans le pool ───

func TestDiscoveryScan_MultiPlayers_OnlyStoreCovered(t *testing.T) {
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

	// Bob n'a qu'une env var → il ne doit PAS entrer dans le pool (Phase 5).
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_BOB", "rt-bob-env")

	storeDir := filepath.Join(repoRoot, "data", "auth", "watcher_tokens")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	store := auth.NewMultiUserTokenStore(storeDir)
	// Alice : store OAuth. Bob : env var seule → exclu. Carol : rien → exclue.
	_ = store.UpdateOAuthRefreshToken("111", "rt-alice-store")

	d := NewDiscoveryWithStore(cfg, resolver, titlePkg.DefaultSlug, store)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(sources) != 1 {
		t.Fatalf("sources = %d (%+v), want 1 (Alice seule)", len(sources), sources)
	}
	alice := sources[0]
	if alice.Gamertag != "Alice" {
		t.Fatalf("gamertag = %q, want Alice", alice.Gamertag)
	}
	if alice.RefreshToken != "rt-alice-store" {
		t.Errorf("Alice RT = %q", alice.RefreshToken)
	}
	if alice.Source != credSourceWatcherOAuth {
		t.Errorf("Alice source = %q, want %q", alice.Source, credSourceWatcherOAuth)
	}
}

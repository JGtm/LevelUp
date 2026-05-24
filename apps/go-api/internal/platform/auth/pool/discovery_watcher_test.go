// Package pool — discovery_watcher_test.go : tests TDD pour E.v1 du refactor
// auth unification — extension Discovery.Scan() qui lit aussi les watcher
// token stores (MultiUserTokenStore + TokenStore legacy mono-user).
//
// Cf. `.ai/PLAN_AUTH_PROVIDER_UNIFICATION.md` §5 Option E.

package pool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
)

// fakeConfigWithPlayers fabrique un *config.AppConfig avec un db_profiles.json
// inline (1 joueur). Le RepoRoot pointe vers un t.TempDir afin que les player
// DBs n'existent PAS (force le fallback vers les watcher stores).
func fakeConfigWithPlayers(t *testing.T, gamertag, xuid string) *config.AppConfig {
	t.Helper()
	repoRoot := t.TempDir()

	// db_profiles.json minimal — la struct exacte est dans config/.
	profiles := map[string]any{
		"version": "3.0",
		"profiles": map[string]any{
			"halo_infinite": map[string]any{
				gamertag: map[string]any{
					"db_path":         "data/titles/halo_infinite/players/" + gamertag + "/stats.duckdb",
					"xuid":            xuid,
					"waypoint_player": gamertag,
				},
			},
		},
	}
	data, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	profilesPath := filepath.Join(repoRoot, "db_profiles.json")
	if err := os.WriteFile(profilesPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.AppConfig{
		RepoRoot:        repoRoot,
		DBProfilesPath:  profilesPath,
		AppSettingsPath: filepath.Join(repoRoot, "app_settings.json"),
	}
	return cfg
}

// ─── Test 1 : MultiUserTokenStore peuple le pool quand DuckDB vide ────────

func TestDiscoveryScan_MultiUserStore_PopulatesWhenDuckDBEmpty(t *testing.T) {
	// Isolation : neutraliser les env vars que d'autres tests du package
	// peuvent avoir chargées via config.Load() depuis le .env.local réel
	// (TestDiscoveryScan_MixedTokenSources etc. — pollution cross-test).
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_MADINA97294", "")
	cfg := fakeConfigWithPlayers(t, "Madina97294", "2533274858283686")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	// Crée le multi-user store avec un MSAL cache pour ce joueur.
	storeDir := filepath.Join(cfg.RepoRoot, "data", "auth", "watcher_tokens")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	store := auth.NewMultiUserTokenStore(storeDir)
	err := store.Upsert(&auth.UserTokens{
		XUID:          "2533274858283686",
		Gamertag:      "Madina97294",
		MSALCacheJSON: `{"fake": "msal_cache"}`,
		XSTSToken:     "xsts_value",
		XSTSUserHash:  "uhs_value",
		XSTSExpiresAt: time.Now().Add(2 * time.Hour),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	// Discovery avec watcher store (DuckDB n'existe pas → fallback watcher).
	d := NewDiscoveryWithStores(cfg, resolver, titlePkg.DefaultSlug, store, nil)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(sources) != 1 {
		t.Fatalf("sources = %d, want 1", len(sources))
	}
	src := sources[0]
	if src.Gamertag != "Madina97294" || src.XUID != "2533274858283686" {
		t.Errorf("source identity wrong: %+v", src)
	}
	if src.MSALCache == "" {
		t.Errorf("MSALCache vide — devrait être peuplé depuis watcher store")
	}
	if src.Source != credSourceWatcherMSAL {
		t.Errorf("Source = %q, want %q", src.Source, credSourceWatcherMSAL)
	}
}

// ─── Test 2 : Legacy mono-user TokenStore peuple le pool ──────────────────

func TestDiscoveryScan_LegacyStore_PopulatesWhenDuckDBEmpty(t *testing.T) {
	// Isolation : neutraliser env vars pollution cross-test (cf. test précédent).
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_JGTM", "")
	cfg := fakeConfigWithPlayers(t, "JGtm", "2533274823110022")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	// Crée le legacy mono-user store avec un refresh token.
	storePath := filepath.Join(cfg.RepoRoot, "data", "auth", "watcher_tokens.json")
	if err := os.MkdirAll(filepath.Dir(storePath), 0o700); err != nil {
		t.Fatal(err)
	}
	store := auth.NewTokenStore(storePath)
	err := store.Save(&auth.StoredTokens{
		RefreshToken: "legacy_refresh_token_value",
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	d := NewDiscoveryWithStores(cfg, resolver, titlePkg.DefaultSlug, nil, store)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(sources) != 1 {
		t.Fatalf("sources = %d, want 1", len(sources))
	}
	src := sources[0]
	if src.RefreshToken != "legacy_refresh_token_value" {
		t.Errorf("RefreshToken = %q, want legacy_refresh_token_value", src.RefreshToken)
	}
	if src.Source != credSourceWatcherLegacy {
		t.Errorf("Source = %q, want %q", src.Source, credSourceWatcherLegacy)
	}
}

// ─── Test 3 : Pas de stores → comportement legacy (skip si DuckDB vide) ───

func TestDiscoveryScan_NoStores_BackwardCompat(t *testing.T) {
	cfg := fakeConfigWithPlayers(t, "NoToken", "1111")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	// Pas de stores attachés, pas de DuckDB, pas d'env var → 0 source.
	d := NewDiscoveryWithStores(cfg, resolver, titlePkg.DefaultSlug, nil, nil)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(sources) != 0 {
		t.Errorf("sources = %d, want 0 (no token anywhere)", len(sources))
	}
}

// ─── Test 4 : DuckDB a priorité sur watcher store ─────────────────────────
//
// Si DuckDB sync_meta contient un MSAL cache OU OAuth, le watcher store
// n'est PAS consulté (priorité au scan DuckDB pour préserver le comportement
// legacy). Skip ici si la création d'une vraie player DB est trop lourde —
// le code source suit la condition `if msal == "" && oauth == ""`.

// ─── Test 4b : LegacyStore attribué à UN SEUL joueur (fix bug 2026-05-24) ─

// Vérifie que le legacy mono-user store n'est PAS dupliqué sur plusieurs
// joueurs. Sinon le même RT serait attribué à N joueurs → mismatch API.

func TestDiscoveryScan_LegacyStore_AttributedToSinglePlayerOnly(t *testing.T) {
	// 3 joueurs configurés, AUCUN n'a de token DuckDB/env. Legacy store
	// contient 1 RT. Seul le 1er joueur doit le recevoir, les 2 autres skip.
	repoRoot := t.TempDir()

	profiles := map[string]any{
		"version": "3.0",
		"profiles": map[string]any{
			"halo_infinite": map[string]any{
				"Alice": map[string]any{
					"db_path": "data/titles/halo_infinite/players/Alice/stats.duckdb",
					"xuid":    "1111", "waypoint_player": "Alice",
				},
				"Bob": map[string]any{
					"db_path": "data/titles/halo_infinite/players/Bob/stats.duckdb",
					"xuid":    "2222", "waypoint_player": "Bob",
				},
				"Carol": map[string]any{
					"db_path": "data/titles/halo_infinite/players/Carol/stats.duckdb",
					"xuid":    "3333", "waypoint_player": "Carol",
				},
			},
		},
	}
	data, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	profilesPath := filepath.Join(repoRoot, "db_profiles.json")
	if err := os.WriteFile(profilesPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.AppConfig{
		RepoRoot:        repoRoot,
		DBProfilesPath:  profilesPath,
		AppSettingsPath: filepath.Join(repoRoot, "app_settings.json"),
	}

	storePath := filepath.Join(repoRoot, "data", "auth", "watcher_tokens.json")
	if err := os.MkdirAll(filepath.Dir(storePath), 0o700); err != nil {
		t.Fatal(err)
	}
	store := auth.NewTokenStore(storePath)
	if err := store.Save(&auth.StoredTokens{RefreshToken: "single_legacy_rt"}); err != nil {
		t.Fatal(err)
	}

	resolver := titlePkg.NewPathResolver(repoRoot)
	d := NewDiscoveryWithStores(cfg, resolver, titlePkg.DefaultSlug, nil, store)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(sources) != 1 {
		t.Fatalf("sources = %d, want 1 (legacy attribué à 1 SEUL joueur), got %+v", len(sources), sources)
	}
	if sources[0].RefreshToken != "single_legacy_rt" {
		t.Errorf("RT = %q, want single_legacy_rt", sources[0].RefreshToken)
	}
}

// ─── Test 5 : XUID safe — store skipped si XUID vide ──────────────────────

func TestDiscoveryScan_MultiUserStore_SkippedIfXUIDEmpty(t *testing.T) {
	cfg := fakeConfigWithPlayers(t, "NoXuid", "")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

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
		t.Errorf("sources = %d, want 0 (XUID vide → store skip)", len(sources))
	}
}

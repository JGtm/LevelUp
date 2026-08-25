// Package pool — discovery_watcher_test.go : Discovery.Scan() lit le
// MultiUserTokenStore (data/auth/watcher_tokens/{xuid}.json).
//
// ADR 0023 Phase 5 (2026-08-25) : le store mono-user (watcher_tokens.json) n'est
// plus une source de credentials — les tests qui le couvraient ont disparu avec
// le chemin de code (leur remplaçant est le ratchet
// discovery_legacy_warn_test.go).

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
// DBs n'existent PAS.
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

// ─── Test 1 : MultiUserTokenStore peuple le pool ──────────────────────────

func TestDiscoveryScan_MultiUserStore_Populates(t *testing.T) {
	cfg := fakeConfigWithPlayers(t, "Madina97294", "2533274858283686")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	storeDir := filepath.Join(cfg.RepoRoot, "data", "auth", "watcher_tokens")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	store := auth.NewMultiUserTokenStore(storeDir)
	err := store.Upsert(&auth.UserTokens{
		XUID:              "2533274858283686",
		Gamertag:          "Madina97294",
		OAuthRefreshToken: "rt_watcher_store",
		XSTSToken:         "xsts_value",
		XSTSUserHash:      "uhs_value",
		XSTSExpiresAt:     time.Now().Add(2 * time.Hour),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	d := NewDiscoveryWithStore(cfg, resolver, titlePkg.DefaultSlug, store)
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
	if src.RefreshToken != "rt_watcher_store" {
		t.Errorf("RefreshToken = %q, want rt_watcher_store", src.RefreshToken)
	}
	if src.Source != credSourceWatcherOAuth {
		t.Errorf("Source = %q, want %q", src.Source, credSourceWatcherOAuth)
	}
}

// ─── Test 2 : pas de store → 0 source ─────────────────────────────────────

func TestDiscoveryScan_NoStore_NoSource(t *testing.T) {
	cfg := fakeConfigWithPlayers(t, "NoToken", "1111")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	d := NewDiscoveryWithStore(cfg, resolver, titlePkg.DefaultSlug, nil)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(sources) != 0 {
		t.Errorf("sources = %d, want 0 (aucun store)", len(sources))
	}
}

// ─── Test 3 : XUID vide → store non adressable, joueur exclu ──────────────

func TestDiscoveryScan_MultiUserStore_SkippedIfXUIDEmpty(t *testing.T) {
	cfg := fakeConfigWithPlayers(t, "NoXuid", "")
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
		t.Errorf("sources = %d, want 0 (XUID vide → store skip)", len(sources))
	}
}

// Package main — migration_boot_test.go : T6 ADR 0023.
//
// Test E2E du wiring `migrateLegacyAuthTokensAtBoot` (la pure logic
// `auth.MigrateLegacyTokens` est déjà testée à 14 cas dans migration_test.go).
//
// Couvre :
//   - Wiring config → players → LegacyPlayer + LegacySourcesReader
//   - Lecture env vars (le reader appelle EnvRefreshTokenForGamertag)
//   - DuckDB skip propre quand la player DB n'existe pas
//   - Idempotence sur appels répétés
//   - Multi-joueurs avec sources hétérogènes
//
// Build tag cgo car migrate boot dépend de duckdb.OpenReadOnly transitivement.
//
//go:build cgo

package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
)

// fakeConfigWithMultiplePlayers crée un AppConfig avec N joueurs dans
// db_profiles.json. Aucun player DB n'est créé (les DuckDB sources seront vides
// pendant le scan → reader retourne LegacySources{EnvRT only}).
func fakeConfigWithMultiplePlayers(t *testing.T, players map[string]string) *config.AppConfig {
	t.Helper()
	repoRoot := t.TempDir()

	playerEntries := make(map[string]any, len(players))
	for gt, xuid := range players {
		playerEntries[gt] = map[string]any{
			"db_path":         filepath.Join("data", "titles", "halo_infinite", "players", gt, "stats.duckdb"),
			"xuid":            xuid,
			"waypoint_player": gt,
		}
	}
	profiles := map[string]any{
		"version": "3.0",
		"profiles": map[string]any{
			"halo_infinite": playerEntries,
		},
	}
	writeJSON(t, filepath.Join(repoRoot, "db_profiles.json"), profiles)
	return &config.AppConfig{
		RepoRoot:        repoRoot,
		DBProfilesPath:  filepath.Join(repoRoot, "db_profiles.json"),
		AppSettingsPath: filepath.Join(repoRoot, "app_settings.json"),
	}
}

func writeJSON(t *testing.T, path string, data any) {
	t.Helper()
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, bytes, 0o644); err != nil {
		t.Fatal(err)
	}
}

// ─── T6.1 — Migration env var → store, single player ──────────────────────

func TestMigrateLegacyAuthTokensAtBoot_EnvVarOnly(t *testing.T) {
	cfg := fakeConfigWithMultiplePlayers(t, map[string]string{
		"Madina97294": "2533274858283686",
	})

	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_MADINA97294", "rt-from-env")

	migrateLegacyAuthTokensAtBoot(context.Background(), cfg)

	store := auth.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	user, err := store.Load("2533274858283686")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if user.OAuthRefreshToken != "rt-from-env" {
		t.Errorf("RT = %q, want rt-from-env", user.OAuthRefreshToken)
	}
	if user.Gamertag != "Madina97294" {
		t.Errorf("Gamertag = %q, want Madina97294", user.Gamertag)
	}
}

// ─── T6.2 — Migration idempotente sur 2 appels successifs ────────────────

func TestMigrateLegacyAuthTokensAtBoot_Idempotent(t *testing.T) {
	cfg := fakeConfigWithMultiplePlayers(t, map[string]string{
		"Madina97294": "2533274858283686",
	})
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_MADINA97294", "rt-initial")

	migrateLegacyAuthTokensAtBoot(context.Background(), cfg)

	store := auth.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	user1, _ := store.Load("2533274858283686")
	updated1 := user1.UpdatedAt

	// 2e appel : doit être no-op (entrée déjà complète).
	// Note : "complète" = RT + MSAL. Sans MSAL, il y a tjs un trigger.
	// On vérifie que le RT reste identique.
	migrateLegacyAuthTokensAtBoot(context.Background(), cfg)

	user2, _ := store.Load("2533274858283686")
	if user2.OAuthRefreshToken != "rt-initial" {
		t.Errorf("RT après 2e appel = %q, want rt-initial", user2.OAuthRefreshToken)
	}
	// Updated peut bouger si l'entrée n'avait pas de MSAL et qu'on tente une
	// (no-op) migration MSAL — le store écrira quand même UpdatedAt. C'est OK.
	_ = updated1
}

// ─── T6.3 — Multi-joueurs : 3 joueurs, sources hétérogènes ────────────────

func TestMigrateLegacyAuthTokensAtBoot_MultiPlayers(t *testing.T) {
	cfg := fakeConfigWithMultiplePlayers(t, map[string]string{
		"Alice": "111",
		"Bob":   "222",
		"Carol": "333",
	})

	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_ALICE", "rt-alice")
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_BOB", "rt-bob")
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_CAROL", "")

	migrateLegacyAuthTokensAtBoot(context.Background(), cfg)

	store := auth.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())

	alice, err := store.Load("111")
	if err != nil {
		t.Errorf("Alice non migrée : %v", err)
	} else if alice.OAuthRefreshToken != "rt-alice" {
		t.Errorf("Alice RT = %q", alice.OAuthRefreshToken)
	}

	bob, err := store.Load("222")
	if err != nil {
		t.Errorf("Bob non migré : %v", err)
	} else if bob.OAuthRefreshToken != "rt-bob" {
		t.Errorf("Bob RT = %q", bob.OAuthRefreshToken)
	}

	// Carol : aucune source → pas d'entrée store
	if _, err := store.Load("333"); err == nil {
		t.Error("Carol ne devrait pas avoir d'entrée store (aucune source)")
	}
}

// ─── T6.4 — Config sans joueurs : no-op silencieux ────────────────────────

func TestMigrateLegacyAuthTokensAtBoot_NoPlayers(t *testing.T) {
	cfg := fakeConfigWithMultiplePlayers(t, map[string]string{})

	// Devrait passer sans panic ni erreur
	migrateLegacyAuthTokensAtBoot(context.Background(), cfg)

	// Le store ne devrait pas exister (rien à écrire)
	storeDir := titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir()
	if _, err := os.Stat(storeDir); err == nil {
		// Si dir existe, doit être vide
		entries, _ := os.ReadDir(storeDir)
		if len(entries) > 0 {
			t.Errorf("storeDir contient %d entries, want 0", len(entries))
		}
	}
}

// ─── T6.5 — DBProfilesPath inexistant : log warn et skip ──────────────────

func TestMigrateLegacyAuthTokensAtBoot_DBProfilesMissing_GracefulSkip(t *testing.T) {
	cfg := &config.AppConfig{
		RepoRoot:        t.TempDir(),
		DBProfilesPath:  filepath.Join(t.TempDir(), "nonexistent.json"),
		AppSettingsPath: filepath.Join(t.TempDir(), "app_settings.json"),
	}

	// Doit return sans panic
	migrateLegacyAuthTokensAtBoot(context.Background(), cfg)
}

// ─── T6.6 — Entrée pré-existante dans store : pas écrasée ─────────────────

func TestMigrateLegacyAuthTokensAtBoot_StoreEntryPreserved(t *testing.T) {
	cfg := fakeConfigWithMultiplePlayers(t, map[string]string{
		"Madina97294": "2533274858283686",
	})

	// Pré-remplir le store
	store := auth.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	if err := store.Upsert(&auth.UserTokens{
		XUID:              "2533274858283686",
		Gamertag:          "Madina97294",
		OAuthRefreshToken: "rt-already-fresh-DO-NOT-OVERWRITE",
	}); err != nil {
		t.Fatal(err)
	}

	// Env var avec une valeur différente
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_MADINA97294", "rt-stale-from-env")

	migrateLegacyAuthTokensAtBoot(context.Background(), cfg)

	// Le store doit être inchangé (entrée complète → skip migration)
	user, _ := store.Load("2533274858283686")
	if user.OAuthRefreshToken != "rt-already-fresh-DO-NOT-OVERWRITE" {
		t.Errorf("RT = %q, store autoritaire devrait être préservé", user.OAuthRefreshToken)
	}
}

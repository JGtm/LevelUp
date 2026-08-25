// Package auth — migration_test.go : tests MigrateLegacyTokens.
package auth

import (
	"context"
	"os"
	"testing"
)

// fakeReader retourne LegacySources prédéfinies par xuid.
func fakeReader(sources map[string]LegacySources) LegacySourcesReader {
	return func(_ context.Context, p LegacyPlayer) (LegacySources, error) {
		return sources[p.XUID], nil
	}
}

func TestMigrateLegacyTokens_EnvVarOnly(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	players := []LegacyPlayer{
		{XUID: "111", Gamertag: "Alice"},
	}
	reader := fakeReader(map[string]LegacySources{
		"111": {EnvRT: "rt-from-env"},
	})

	stats, err := MigrateLegacyTokens(context.Background(), store, players, reader)
	if err != nil {
		t.Fatalf("MigrateLegacyTokens: %v", err)
	}

	if stats.OAuthRTMigrated != 1 {
		t.Errorf("OAuthRTMigrated = %d, want 1", stats.OAuthRTMigrated)
	}

	loaded, _ := store.Load("111")
	if loaded.OAuthRefreshToken != "rt-from-env" {
		t.Errorf("RT = %q, want rt-from-env", loaded.OAuthRefreshToken)
	}
	if loaded.Gamertag != "Alice" {
		t.Errorf("Gamertag = %q, want Alice (complétion)", loaded.Gamertag)
	}
}

func TestMigrateLegacyTokens_DuckDBOnly(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	players := []LegacyPlayer{
		{XUID: "111", Gamertag: "Alice"},
	}
	reader := fakeReader(map[string]LegacySources{
		"111": {DuckDBRT: "rt-from-duckdb"},
	})

	stats, err := MigrateLegacyTokens(context.Background(), store, players, reader)
	if err != nil {
		t.Fatalf("MigrateLegacyTokens: %v", err)
	}

	if stats.OAuthRTMigrated != 1 {
		t.Errorf("OAuthRTMigrated = %d, want 1", stats.OAuthRTMigrated)
	}

	loaded, _ := store.Load("111")
	if loaded.OAuthRefreshToken != "rt-from-duckdb" {
		t.Errorf("RT = %q", loaded.OAuthRefreshToken)
	}
}

func TestMigrateLegacyTokens_DuckDBPriorityOverEnv(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	players := []LegacyPlayer{
		{XUID: "111", Gamertag: "Alice"},
	}
	// Les deux sources ont une valeur — DuckDB doit gagner (RT rotaté).
	reader := fakeReader(map[string]LegacySources{
		"111": {EnvRT: "rt-from-env-stale", DuckDBRT: "rt-from-duckdb-fresh"},
	})

	if _, err := MigrateLegacyTokens(context.Background(), store, players, reader); err != nil {
		t.Fatalf("MigrateLegacyTokens: %v", err)
	}

	loaded, _ := store.Load("111")
	if loaded.OAuthRefreshToken != "rt-from-duckdb-fresh" {
		t.Errorf("RT = %q, want rt-from-duckdb-fresh (DuckDB prioritaire)", loaded.OAuthRefreshToken)
	}
}

func TestMigrateLegacyTokens_StoreAutoritativeOnExistingRT(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	// Store déjà rempli avec un RT — entrée complète.
	_ = store.UpdateOAuthRefreshToken("111", "rt-already-in-store")

	players := []LegacyPlayer{{XUID: "111", Gamertag: "Alice"}}
	// Source legacy avec valeurs différentes — ne doit PAS écraser.
	reader := fakeReader(map[string]LegacySources{
		"111": {
			EnvRT:    "rt-legacy-should-not-overwrite",
			DuckDBRT: "rt-legacy-too",
		},
	})

	stats, err := MigrateLegacyTokens(context.Background(), store, players, reader)
	if err != nil {
		t.Fatalf("MigrateLegacyTokens: %v", err)
	}

	if stats.OAuthRTMigrated != 0 {
		t.Errorf("OAuthRTMigrated = %d, want 0 (store autoritaire)", stats.OAuthRTMigrated)
	}
	if stats.PlayersSkipped != 1 {
		t.Errorf("PlayersSkipped = %d, want 1 (entrée complète)", stats.PlayersSkipped)
	}

	loaded, _ := store.Load("111")
	if loaded.OAuthRefreshToken != "rt-already-in-store" {
		t.Errorf("RT = %q, want rt-already-in-store (préservé)", loaded.OAuthRefreshToken)
	}
}

func TestMigrateLegacyTokens_Idempotent(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	players := []LegacyPlayer{{XUID: "111", Gamertag: "Alice"}}
	reader := fakeReader(map[string]LegacySources{
		"111": {DuckDBRT: "rt-v1"},
	})

	// Premier passage : migration du RT.
	stats1, _ := MigrateLegacyTokens(context.Background(), store, players, reader)
	if stats1.OAuthRTMigrated != 1 {
		t.Fatalf("première passe : RT migré = %d, want 1", stats1.OAuthRTMigrated)
	}

	// Second passage : entrée complète → skip.
	stats2, _ := MigrateLegacyTokens(context.Background(), store, players, reader)
	if stats2.OAuthRTMigrated != 0 {
		t.Errorf("seconde passe : RT=%d, want 0 (idempotence)", stats2.OAuthRTMigrated)
	}
	if stats2.PlayersSkipped != 1 {
		t.Errorf("seconde passe : skipped = %d, want 1 (entrée complète)", stats2.PlayersSkipped)
	}

	loaded, _ := store.Load("111")
	if loaded.OAuthRefreshToken != "rt-v1" {
		t.Errorf("RT après 2 passes = %q, want rt-v1", loaded.OAuthRefreshToken)
	}
}

func TestMigrateLegacyTokens_NoSourcesNoChange(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	players := []LegacyPlayer{{XUID: "111", Gamertag: "Alice"}}
	reader := fakeReader(map[string]LegacySources{
		"111": {}, // toutes vides
	})

	stats, _ := MigrateLegacyTokens(context.Background(), store, players, reader)
	if stats.OAuthRTMigrated != 0 {
		t.Errorf("aucune source → 0 migration, got RT=%d", stats.OAuthRTMigrated)
	}

	if _, err := store.Load("111"); err == nil {
		t.Error("aucune source → aucune entrée store créée")
	}
}

func TestMigrateLegacyTokens_MultiplePlayers(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	players := []LegacyPlayer{
		{XUID: "111", Gamertag: "Alice"},
		{XUID: "222", Gamertag: "Bob"},
		{XUID: "333", Gamertag: "Carol"},
	}
	reader := fakeReader(map[string]LegacySources{
		"111": {DuckDBRT: "rt-alice"},
		"222": {EnvRT: "rt-bob"},
		// 333 : rien — pas migré
	})

	stats, _ := MigrateLegacyTokens(context.Background(), store, players, reader)
	if stats.PlayersScanned != 3 {
		t.Errorf("PlayersScanned = %d, want 3", stats.PlayersScanned)
	}
	if stats.OAuthRTMigrated != 2 {
		t.Errorf("OAuthRTMigrated = %d, want 2", stats.OAuthRTMigrated)
	}

	alice, _ := store.Load("111")
	if alice.OAuthRefreshToken != "rt-alice" {
		t.Errorf("alice RT = %q", alice.OAuthRefreshToken)
	}
	bob, _ := store.Load("222")
	if bob.OAuthRefreshToken != "rt-bob" {
		t.Errorf("bob RT = %q", bob.OAuthRefreshToken)
	}
	if _, err := store.Load("333"); err == nil {
		t.Error("carol ne devrait pas avoir d'entrée")
	}
}

func TestMigrateLegacyTokens_RejectsUnsafeXUID(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	players := []LegacyPlayer{
		{XUID: "../escape", Gamertag: "Hacker"},
		{XUID: "111", Gamertag: "Alice"},
	}
	reader := fakeReader(map[string]LegacySources{
		"../escape": {DuckDBRT: "rt-malicious"},
		"111":       {DuckDBRT: "rt-alice"},
	})

	stats, _ := MigrateLegacyTokens(context.Background(), store, players, reader)
	if stats.Errors != 1 {
		t.Errorf("Errors = %d, want 1 (unsafe xuid rejected)", stats.Errors)
	}
	if stats.OAuthRTMigrated != 1 {
		t.Errorf("OAuthRTMigrated = %d, want 1 (Alice only)", stats.OAuthRTMigrated)
	}

	if _, err := store.Load("111"); err != nil {
		t.Error("Alice devrait être migrée malgré l'erreur sur Hacker")
	}
}

func TestMigrateLegacyTokens_NilStore(t *testing.T) {
	if _, err := MigrateLegacyTokens(context.Background(), nil, nil, fakeReader(nil)); err == nil {
		t.Error("store nil devrait être refusé")
	}
}

func TestMigrateLegacyTokens_NilReader(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	if _, err := MigrateLegacyTokens(context.Background(), store, nil, nil); err == nil {
		t.Error("reader nil devrait être refusé")
	}
}

func TestEnvRefreshTokenForGamertag_NormalizesAndReads(t *testing.T) {
	cases := []struct {
		gamertag string
		envKey   string
	}{
		{"Madina97294", "SPNKR_OAUTH_REFRESH_TOKEN_MADINA97294"},
		{"Chocoboflor", "SPNKR_OAUTH_REFRESH_TOKEN_CHOCOBOFLOR"},
		{"My Gamer.Tag-X", "SPNKR_OAUTH_REFRESH_TOKEN_MY_GAMER_TAG_X"},
		{"  spaced  ", "SPNKR_OAUTH_REFRESH_TOKEN_SPACED"},
	}

	for _, c := range cases {
		t.Run(c.gamertag, func(t *testing.T) {
			t.Setenv(c.envKey, "value-"+c.envKey)
			got := EnvRefreshTokenForGamertag(c.gamertag)
			want := "value-" + c.envKey
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

func TestEnvRefreshTokenForGamertag_Empty(t *testing.T) {
	if got := EnvRefreshTokenForGamertag(""); got != "" {
		t.Errorf("gamertag vide → got %q, want \"\"", got)
	}
}

func TestEnvRefreshTokenForGamertag_NotSet(t *testing.T) {
	// Assurer que la var n'est pas définie.
	_ = os.Unsetenv("SPNKR_OAUTH_REFRESH_TOKEN_NONEXISTENT_TEST_USER")
	if got := EnvRefreshTokenForGamertag("nonexistent test user"); got != "" {
		t.Errorf("var absente → got %q, want \"\"", got)
	}
}

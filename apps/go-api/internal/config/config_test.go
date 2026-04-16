package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestLoadPlayers_V3_TitleIsolation vérifie que les joueurs de chaque titre
// sont correctement séparés lors du chargement d'un db_profiles.json v3.
func TestLoadPlayers_V3_TitleIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	profilesPath := filepath.Join(tmpDir, "db_profiles.json")

	// Fichier v3 avec deux titres
	v3 := dbProfilesFileV3{
		Version: "3.0",
		Profiles: map[string]map[string]dbProfileEntry{
			"halo_infinite": {
				"PlayerHI": {
					DBPath:         "data/titles/halo_infinite/players/PlayerHI/stats.duckdb",
					XUID:           "1111111111111111",
					WaypointPlayer: "PlayerHI",
				},
			},
			"halo_mcc": {
				"PlayerMCC": {
					DBPath:         "data/titles/halo_mcc/players/PlayerMCC/stats.duckdb",
					XUID:           "2222222222222222",
					WaypointPlayer: "PlayerMCC",
				},
			},
		},
	}
	data, err := json.MarshalIndent(v3, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(profilesPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &AppConfig{
		RepoRoot:       tmpDir,
		DBProfilesPath: profilesPath,
	}

	// Sans filtre : tous les joueurs
	all, err := cfg.LoadPlayers()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 players total, got %d", len(all))
	}

	// Filtre halo_infinite : uniquement PlayerHI
	hi, err := cfg.LoadPlayers("halo_infinite")
	if err != nil {
		t.Fatal(err)
	}
	if len(hi) != 1 {
		t.Fatalf("expected 1 halo_infinite player, got %d", len(hi))
	}
	if hi[0].Gamertag != "PlayerHI" {
		t.Errorf("expected PlayerHI, got %q", hi[0].Gamertag)
	}

	// Filtre halo_mcc : uniquement PlayerMCC
	mcc, err := cfg.LoadPlayers("halo_mcc")
	if err != nil {
		t.Fatal(err)
	}
	if len(mcc) != 1 {
		t.Fatalf("expected 1 halo_mcc player, got %d", len(mcc))
	}
	if mcc[0].Gamertag != "PlayerMCC" {
		t.Errorf("expected PlayerMCC, got %q", mcc[0].Gamertag)
	}
}

// TestLoadPlayers_V2_BackwardCompatible vérifie qu'un fichier v2.1 est
// lu correctement et que tous les profils sont implicitement halo_infinite.
func TestLoadPlayers_V2_BackwardCompatible(t *testing.T) {
	tmpDir := t.TempDir()
	profilesPath := filepath.Join(tmpDir, "db_profiles.json")

	v2 := dbProfilesFile{
		Version: "2.1",
		Profiles: map[string]dbProfileEntry{
			"LegacyPlayer": {
				DBPath:         "data/players/LegacyPlayer/stats.duckdb",
				XUID:           "3333333333333333",
				WaypointPlayer: "LegacyPlayer",
			},
		},
	}
	data, err := json.MarshalIndent(v2, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(profilesPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &AppConfig{
		RepoRoot:       tmpDir,
		DBProfilesPath: profilesPath,
	}

	// Sans filtre
	all, err := cfg.LoadPlayers()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 player, got %d", len(all))
	}
	if all[0].Gamertag != "LegacyPlayer" {
		t.Errorf("expected LegacyPlayer, got %q", all[0].Gamertag)
	}

	// Filtre halo_infinite : doit retourner le joueur v2
	hi, err := cfg.LoadPlayers("halo_infinite")
	if err != nil {
		t.Fatal(err)
	}
	if len(hi) != 1 {
		t.Errorf("expected 1 player for halo_infinite filter, got %d", len(hi))
	}

	// Filtre halo_mcc : rien
	mcc, err := cfg.LoadPlayers("halo_mcc")
	if err != nil {
		t.Fatal(err)
	}
	if len(mcc) != 0 {
		t.Errorf("expected 0 players for halo_mcc filter on v2 file, got %d", len(mcc))
	}
}

// TestLoadPlayers_V3_EmptyTitle vérifie qu'un titre vide dans v3 ne plante pas.
func TestLoadPlayers_V3_EmptyTitle(t *testing.T) {
	tmpDir := t.TempDir()
	profilesPath := filepath.Join(tmpDir, "db_profiles.json")

	v3 := dbProfilesFileV3{
		Version: "3.0",
		Profiles: map[string]map[string]dbProfileEntry{
			"halo_infinite": {},
			"halo_mcc":      {},
		},
	}
	data, err := json.MarshalIndent(v3, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(profilesPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &AppConfig{
		RepoRoot:       tmpDir,
		DBProfilesPath: profilesPath,
	}

	players, err := cfg.LoadPlayers()
	if err != nil {
		t.Fatal(err)
	}
	if len(players) != 0 {
		t.Errorf("expected 0 players for empty titles, got %d", len(players))
	}
}

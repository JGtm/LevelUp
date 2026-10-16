package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

func TestProfileService_CreatePlayer_NewFile(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "db_profiles.json")
	repoRoot := dir

	svc := NewProfileService(dbPath, repoRoot)

	key, warnings, err := svc.CreatePlayer(domain.CreatePlayerProfileRequest{
		Gamertag: "TestPlayer",
		XUID:     "xuid123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key == "" {
		t.Error("expected non-empty key")
	}
	_ = warnings

	// File should exist
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected db_profiles.json to be created")
	}
}

func TestProfileService_CreatePlayer_ExistingV3(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "db_profiles.json")
	repoRoot := dir

	// Write a v3 file
	initial := `{"version":"3.0","profiles":{"halo-infinite":{"testplayer":{"db_path":"data/players/testplayer/stats.duckdb","xuid":"x1"}}}}`
	if err := os.WriteFile(dbPath, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	svc := NewProfileService(dbPath, repoRoot)

	key, _, err := svc.CreatePlayer(domain.CreatePlayerProfileRequest{
		Gamertag: "NewPlayer",
		XUID:     "x2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key == "" {
		t.Error("expected non-empty key")
	}
}

func TestProfileService_CreatePlayer_MigrateV2(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "db_profiles.json")

	// Write a v2.1 file
	data := `{"version":"2.1","profiles":{"player1":{"db_path":"p1","xuid":"x1"}}}`
	if err := os.WriteFile(dbPath, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	svc := NewProfileService(dbPath, dir)
	key, _, err := svc.CreatePlayer(domain.CreatePlayerProfileRequest{
		Gamertag: "Player2",
		XUID:     "x2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key == "" {
		t.Error("expected non-empty key")
	}
}

// TestProfileService_CreatePlayer_RefusesXUIDChangeOfExistingProfile : la propriété
// d'un profil suit son xuid (ADR 0029) ; le réécrire sans droit (admin) transférerait
// le profil à un autre compte (registre sécurité C4-5).
func TestProfileService_CreatePlayer_RefusesXUIDChangeOfExistingProfile(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "db_profiles.json")
	initial := `{"version":"3.0","profiles":{"halo_infinite":{"Victim":{"db_path":"data/players/Victim/stats.duckdb","xuid":"999"}}}}`
	if err := os.WriteFile(dbPath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewProfileService(dbPath, dir)
	xuidOf := func() string {
		players, err := (&config.AppConfig{DBProfilesPath: dbPath}).LoadPlayers("halo_infinite")
		if err != nil || len(players) != 1 {
			t.Fatalf("LoadPlayers: %v (%d)", err, len(players))
		}
		return players[0].XUID
	}

	_, _, err := svc.CreatePlayer(domain.CreatePlayerProfileRequest{Gamertag: "victim", XUID: "222", TitleSlug: "halo_infinite"})
	if !errors.Is(err, domain.ErrProfileXUIDConflict) {
		t.Fatalf("err = %v, attendu ErrProfileXUIDConflict", err)
	}
	if got := xuidOf(); got != "999" {
		t.Fatalf("xuid réécrit en %q malgré le refus", got)
	}

	// Même xuid, ou xuid absent de la demande : mise à jour acceptée.
	if _, _, err := svc.CreatePlayer(domain.CreatePlayerProfileRequest{Gamertag: "Victim", XUID: "999", TitleSlug: "halo_infinite"}); err != nil {
		t.Fatalf("même xuid : %v", err)
	}
	// Admin : changement permis.
	if _, _, err := svc.CreatePlayer(domain.CreatePlayerProfileRequest{Gamertag: "Victim", XUID: "222",
		TitleSlug: "halo_infinite", AllowXUIDChange: true}); err != nil {
		t.Fatalf("admin : %v", err)
	}
	if got := xuidOf(); got != "222" {
		t.Fatalf("admin : xuid %q, attendu 222", got)
	}
}

package service

import (
	"os"
	"path/filepath"
	"testing"

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

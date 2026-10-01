package api

// server_player_directory_demo_test.go — en démo, la section Identités ne balaie pas les
// dossiers de joueurs du dépôt (revue adversariale du lot B5, constat C3 ; lot B-C2 du
// backlog 2026-09-26).
//
// Avant : buildPlayerDirectory construisait le témoin disque sur cfg.RepoRoot, même en démo.
// Sur un poste de dev (harnais visuel : LEVELUP_REPO_ROOT = le vrai checkout), la démo
// publique listait les vrais dossiers de joueurs du poste comme « dossiers orphelins ».

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
)

// depotLeurre : un dépôt avec le dossier d'un VRAI joueur, qu'aucun profil ne déclare, et
// une fixture démo sans profil.
func depotLeurre(t *testing.T, demo bool) *config.AppConfig {
	t.Helper()
	repo := t.TempDir()
	playerDir := title.NewPathResolver(repo).PlayerDir(title.DefaultSlug, "VraiJoueur")
	if err := os.MkdirAll(playerDir, 0o755); err != nil {
		t.Fatalf("dossier joueur réel : %v", err)
	}
	if err := os.WriteFile(filepath.Join(playerDir, "stats.duckdb"), []byte("x"), 0o644); err != nil {
		t.Fatalf("sentinelle : %v", err)
	}
	fixture := t.TempDir()
	profiles := filepath.Join(fixture, "db_profiles.json")
	if err := os.WriteFile(profiles, []byte(`{"version":"3.0","profiles":{}}`), 0o644); err != nil {
		t.Fatalf("db_profiles : %v", err)
	}
	return &config.AppConfig{DemoMode: demo, RepoRoot: repo, DemoFixturesDir: fixture, DBProfilesPath: profiles}
}

func identitesListees(t *testing.T, cfg *config.AppConfig) []string {
	t.Helper()
	resp, err := buildPlayerDirectory(playerDirectoryDeps{cfg: cfg}).List(context.Background())
	if err != nil {
		t.Fatalf("List : %v", err)
	}
	var gamertags []string
	for _, rec := range resp.Identities {
		gamertags = append(gamertags, rec.Gamertag)
	}
	return gamertags
}

func TestPlayerDirectory_Demo_NeBalaiePasLeDepot(t *testing.T) {
	for _, gt := range identitesListees(t, depotLeurre(t, true)) {
		if gt == "VraiJoueur" {
			t.Fatalf("démo : le dossier d'un joueur réel du dépôt apparaît dans les identités (%v)", gt)
		}
	}
}

func TestPlayerDirectory_HorsDemo_ListeLesOrphelins(t *testing.T) {
	gts := identitesListees(t, depotLeurre(t, false))
	for _, gt := range gts {
		if gt == "VraiJoueur" {
			return
		}
	}
	t.Fatalf("hors démo : le dossier orphelin du dépôt devait apparaître (comportement inchangé), identités = %v", gts)
}

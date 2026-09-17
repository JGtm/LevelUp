// Package scheduler — auto_sync_pool_gate_test.go : le cycle d'auto-sync ne saute plus un
// profil suivi qui n'a pas SON propre token.
//
// POURQUOI (2026-09-16). `checkSyncPreconditions` portait une précondition
// `pool.HasPlayer(gamertag)` : un profil suivi sans refresh token propre était sauté à VIE par
// le cycle (`joueur absent du pool`), alors que l'historique, les stats, les films et les CSR
// sont des endpoints PUBLICS servis par n'importe quel token du parc (PolicyAnyPublic). Seul
// le rang de carrière lui est inaccessible, et il se dégrade seul.
//
// Test INTERNE (package scheduler) : `checkSyncPreconditions` n'est pas exportée.
package scheduler

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	settings_platform "levelup/go-api/internal/platform/settings"
)

// schedulerAvecPool : scheduler isolé (repoRoot temp, settings minimaux) sur le pool donné.
func schedulerAvecPool(t *testing.T, repoRoot string, has map[string]bool, avecPool bool) *AutoSyncScheduler {
	t.Helper()
	settingsPath := filepath.Join(repoRoot, "app_settings.json")
	if err := os.WriteFile(settingsPath,
		[]byte(`{"spnkr_auto_sync_enabled":true,"spnkr_auto_sync_interval_hours":1}`), 0o644); err != nil {
		t.Fatalf("écriture settings: %v", err)
	}
	cfg := &config.AppConfig{RepoRoot: repoRoot, AppSettingsPath: settingsPath}
	if !avecPool {
		return New(cfg, settings_platform.NewStore(settingsPath), nil, nil)
	}
	return New(cfg, settings_platform.NewStore(settingsPath), nil, &h5FakePool{has: has})
}

// creerPlayerDB pose un fichier stats.duckdb VIDE au chemin attendu : la précondition de
// présence est un os.Stat, elle n'ouvre jamais la base (aucune DB n'est ouverte par ce test).
func creerPlayerDB(t *testing.T, repoRoot, gamertag string) {
	t.Helper()
	dir := filepath.Join(repoRoot, "data", "titles", "halo_infinite", "players", gamertag)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir player dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "stats.duckdb"), []byte{}, 0o644); err != nil {
		t.Fatalf("écriture stats.duckdb: %v", err)
	}
}

// TestPreconditions_JoueurHorsPoolAccepte — LE test de la décision D1. Il ROUGIT si la
// précondition `HasPlayer` revient (vérifié le 2026-09-16 en la remettant).
func TestPreconditions_JoueurHorsPoolAccepte(t *testing.T) {
	repoRoot := t.TempDir()
	creerPlayerDB(t, repoRoot, "Nuzzles")
	// Le pool ne contient QUE JGtm — Nuzzles n'a pas de token propre.
	s := schedulerAvecPool(t, repoRoot, map[string]bool{"JGtm": true}, true)

	raison, ok := s.checkSyncPreconditions(context.Background(),
		domain.PlayerSummary{Gamertag: "Nuzzles", XUID: "2533274800000000", TitleSlug: "halo_infinite"})
	if !ok {
		t.Fatalf("un profil suivi sans token propre doit être synchronisé par le pool (D1, "+
			"plan 2026-09-16) — skip=%q", raison)
	}
}

// TestPreconditions_PoolNilRefuse — la précondition qui RESTE : sans aucun token découvert,
// il n'y a plus personne pour parler à l'API.
func TestPreconditions_PoolNilRefuse(t *testing.T) {
	repoRoot := t.TempDir()
	creerPlayerDB(t, repoRoot, "Nuzzles")
	s := schedulerAvecPool(t, repoRoot, nil, false)

	raison, ok := s.checkSyncPreconditions(context.Background(),
		domain.PlayerSummary{Gamertag: "Nuzzles", XUID: "2533274800000000", TitleSlug: "halo_infinite"})
	if ok {
		t.Fatal("pool nil : la précondition doit refuser (aucun credential découvert au boot)")
	}
	if raison == "" {
		t.Error("raison de skip vide alors que le pool est nil")
	}
}

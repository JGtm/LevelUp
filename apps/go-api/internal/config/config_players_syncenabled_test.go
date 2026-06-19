package config

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
)

// loadPlayersV3 NE DOIT PAS filtrer les titres en pause : le même loader alimente
// l'UI (bootstrap, liste /players, résolution de page). Filtrer ici provoquerait
// un 404 sur les pages d'un joueur en pause et rendrait la réactivation impossible.
// Le filtre sync est appliqué en aval, dans les chemins SYNC uniquement
// (domain.SyncablePlayers).
func TestLoadPlayersV3_KeepsPausedPlayer_AndResolvesSyncEnabled(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "db_profiles.json")
	content := `{"version":"3.0","profiles":{
		"halo_infinite":{"JGtm":{"db_path":"d","xuid":"1"}},
		"halo_5":{"JGtm":{"db_path":"d5","xuid":"1","sync_enabled":false,"initial_max_matches":50}}
	}}`
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &AppConfig{DBProfilesPath: p}

	players, err := cfg.LoadPlayers()
	if err != nil {
		t.Fatalf("LoadPlayers: %v", err)
	}
	if len(players) != 2 {
		t.Fatalf("attendu 2 couples (le titre en pause reste VISIBLE pour l'UI), reçu %d", len(players))
	}

	byTitle := map[string]domain.PlayerSummary{}
	for _, pl := range players {
		byTitle[pl.TitleSlug] = pl
	}
	if hi, ok := byTitle["halo_infinite"]; !ok || !hi.SyncEnabled {
		t.Fatalf("halo_infinite : champ absent ou SyncEnabled résolu faux (nil→true attendu): %+v", hi)
	}
	h5, ok := byTitle["halo_5"]
	if !ok {
		t.Fatalf("le couple halo_5 en pause a disparu du loader")
	}
	if h5.SyncEnabled {
		t.Fatalf("halo_5 devrait être SyncEnabled=false")
	}
	if h5.InitialMaxMatches != 50 {
		t.Fatalf("InitialMaxMatches non propagé: %d", h5.InitialMaxMatches)
	}

	// Le filtre sync (aval) ne garde que le titre actif.
	syncable := domain.SyncablePlayers(players)
	if len(syncable) != 1 || syncable[0].TitleSlug != "halo_infinite" {
		t.Fatalf("SyncablePlayers devrait ne garder que halo_infinite, reçu %+v", syncable)
	}
}

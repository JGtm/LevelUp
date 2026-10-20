package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

// creerBaseDeTest crée un fichier DuckDB et y joue les instructions.
func creerBaseDeTest(t *testing.T, path string, stmts ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer db.Close()
	for _, s := range stmts {
		if _, err := db.Exec(context.Background(), s); err != nil {
			t.Fatalf("exec %.50s: %v", s, err)
		}
	}
}

func lireMapName(t *testing.T, path, matchID string) string {
	t.Helper()
	db, release, err := duckdb.OpenReadForQuery(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer release()
	var nom *string
	if err := db.QueryRow(`SELECT map_name FROM match_registry WHERE match_id = ?`, matchID).Scan(&nom); err != nil {
		t.Fatalf("lecture: %v", err)
	}
	if nom == nil {
		return "<NULL>"
	}
	return *nom
}

// TestBackfillRegistryNames_Commande : --dry-run n'écrit rien ; la passe réelle réécrit NULL et
// nom = identifiant ; une seconde passe est sans effet.
func TestBackfillRegistryNames_Commande(t *testing.T) {
	repoRoot := t.TempDir()
	pr := titlePkg.NewPathResolver(repoRoot)
	sharedPath := pr.SharedDBPath(titlePkg.DefaultSlug)
	creerBaseDeTest(t, sharedPath,
		`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY,
			playlist_id VARCHAR, playlist_name VARCHAR, map_id VARCHAR, map_name VARCHAR,
			pair_id VARCHAR, pair_name VARCHAR, game_variant_id VARCHAR, game_variant_name VARCHAR,
			mode_category VARCHAR)`,
		`INSERT INTO match_registry (match_id, map_id, map_name) VALUES
			('m1', 'map-x', NULL), ('m2', 'map-x', 'map-x'), ('m3', 'map-x', 'Streets')`)
	creerBaseDeTest(t, pr.MetadataDBPath(titlePkg.DefaultSlug),
		`CREATE TABLE asset_translations (asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR)`,
		`INSERT INTO asset_translations VALUES ('map-x', 'map', 'en-US', 'Streets')`)
	cfg := &config.AppConfig{RepoRoot: repoRoot}

	if err := runBackfillRegistryNames(cfg, []string{"--dry-run"}); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if got := lireMapName(t, sharedPath, "m1"); got != "<NULL>" {
		t.Fatalf("--dry-run a écrit : m1 map_name = %q", got)
	}
	for i := 0; i < 2; i++ { // seconde passe : idempotente
		if err := runBackfillRegistryNames(cfg, nil); err != nil {
			t.Fatalf("passe %d: %v", i+1, err)
		}
	}
	for _, id := range []string{"m1", "m2", "m3"} {
		if got := lireMapName(t, sharedPath, id); got != "Streets" {
			t.Errorf("%s map_name = %q, want Streets", id, got)
		}
	}
}

//go:build cgo

package main

import (
	"errors"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
)

// TestBackfillRegistryNames_BaseTenueParUnAutreProcessus : un AUTRE PROCESSUS tient une base de
// la commande ouverte en écriture (ce que fait le serveur : metadata.duckdb en permanence, la base
// partagée pendant ses écritures). La commande échoue sur errServeurLance — en simulation comme
// en écriture — sans rien écrire.
func TestBackfillRegistryNames_BaseTenueParUnAutreProcessus(t *testing.T) {
	repoRoot := t.TempDir()
	pr := titlePkg.NewPathResolver(repoRoot)
	sharedPath := pr.SharedDBPath(titlePkg.DefaultSlug)
	metaPath := pr.MetadataDBPath(titlePkg.DefaultSlug)
	creerBaseDeTest(t, sharedPath,
		`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY,
			playlist_id VARCHAR, playlist_name VARCHAR, map_id VARCHAR, map_name VARCHAR,
			pair_id VARCHAR, pair_name VARCHAR, game_variant_id VARCHAR, game_variant_name VARCHAR,
			mode_category VARCHAR)`,
		`INSERT INTO match_registry (match_id, map_id, map_name) VALUES ('m1', 'map-x', NULL)`)
	creerBaseDeTest(t, metaPath,
		`CREATE TABLE asset_translations (asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR)`,
		`INSERT INTO asset_translations VALUES ('map-x', 'map', 'en-US', 'Streets')`)
	cfg := &config.AppConfig{RepoRoot: repoRoot}

	for _, tenue := range []struct{ nom, path string }{{"métadonnées", metaPath}, {"partagée", sharedPath}} {
		t.Run(tenue.nom, func(t *testing.T) {
			liberer := tenirLaBase(t, tenue.path)
			for _, args := range [][]string{{"--dry-run"}, nil} {
				if err := runBackfillRegistryNames(cfg, args); !errors.Is(err, errServeurLance) {
					t.Errorf("args %v : err = %v, want errServeurLance", args, err)
				}
			}
			liberer()
		})
	}
	if got := lireMapName(t, sharedPath, "m1"); got != "<NULL>" {
		t.Errorf("m1 map_name = %q, want NULL (aucune écriture)", got)
	}
}

//go:build integration

package migrations_test

// migration_isolation_test.go — oracle d'ISOLATION : avec le provider Halo câblé
// (comme en production), RunForTitleDB(halo_5, <target>) applique le set VIDE de
// Halo 5 et JAMAIS les migrations Halo Infinite. Sans cet enregistrement, le
// runner retomberait sur le fallback legacy (registre global Halo) et créerait
// des tables match_registry/etc. parasites dans le warehouse du titre live-only.

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halo5 "levelup/go-api/internal/games/halo_5"
	h5migrations "levelup/go-api/internal/games/halo_5/migrations"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", name,
	).Scan(&n); err != nil {
		t.Fatalf("tableExists(%s): %v", name, err)
	}
	return n == 1
}

// TestHalo5_MigrationIsolation : sur chaque target provisionné pour un titre
// additionnel, le set vide de Halo 5 dégrade en no-op propre et AUCUNE table Halo
// Infinite ne fuite dans le warehouse h5.
func TestHalo5_MigrationIsolation(t *testing.T) {
	// Provider Halo câblé comme en prod : prouve que le routage par set (map)
	// l'emporte et que halo_5 n'hérite JAMAIS du registre global Halo.
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	h5migrations.Register()

	// Les targets provisionnés pour un titre additionnel (cf. provisionAdditionalTitle).
	targets := []migration.TargetDB{
		migration.TargetMetadata,
		migration.TargetShared,
		migration.TargetSharedSocial,
		migration.TargetSharedPvE,
	}
	// Tables/vues signature de Halo Infinite : leur présence = fuite du registre global.
	haloArtifacts := []string{
		"match_registry", "match_participants", "medals_earned",
		"v_gamertag_lookup", "asset_translations", "career_rank_translations",
	}

	for _, target := range targets {
		db, err := sql.Open("duckdb", ":memory:")
		if err != nil {
			t.Fatalf("open duckdb: %v", err)
		}
		if err := migration.RunForTitleDB(db, halo5.TitleSlug, target); err != nil {
			db.Close()
			t.Fatalf("RunForTitleDB(%s, %s) doit être un no-op propre: %v", halo5.TitleSlug, target, err)
		}
		for _, artifact := range haloArtifacts {
			if tableExists(t, db, artifact) {
				t.Errorf("target %s : artefact Halo %q présent dans le warehouse halo_5 — isolation cassée", target, artifact)
			}
		}
		db.Close()
	}
}

//go:build integration

// milestones_test.go — déplacé depuis internal/migration (Phase 1.5 b11) avec
// seedMilestonesFromTOML/seedMilestonesAllTitles/RegisterMilestonesSeedMigration
// (milestones.go). Ce package peut câbler le provider title-owned (StepsFor) pour
// faire tourner RunForDB(TargetMetadata) bout-en-bout.
package migrations

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// configTitlesRootHelper retourne config/titles/ depuis
// apps/go-api/internal/games/halo_infinite/migrations (6 niveaux jusqu'à la racine).
func configTitlesRootHelper(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", "..", "config", "titles"))
	if err != nil {
		t.Fatalf("configTitlesRoot abs: %v", err)
	}
	return abs
}

// openMetadataDB ouvre une DB :memory: et applique toutes les migrations metadata
// (global + title-owned via le provider) — crée notamment milestone_catalog.
func openMetadataDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForDB(Metadata): %v", err)
	}
	return db
}

// TestSeedMilestonesFromTOML_HaloInfinite_PopulatesRows : milestones d'Halo
// Infinite chargés depuis le vrai TOML config.
func TestSeedMilestonesFromTOML_HaloInfinite_PopulatesRows(t *testing.T) {
	db := openMetadataDB(t)

	path := filepath.Join(configTitlesRootHelper(t), "halo_infinite", "milestones", "catalog.toml")
	if err := seedMilestonesFromTOML(db, path); err != nil {
		t.Fatalf("seedMilestonesFromTOML: %v", err)
	}

	var count int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM milestone_catalog WHERE title_slug = 'halo_infinite'`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected >0 milestones for halo_infinite, got 0 — TOML vide ou parse échoué")
	}
	var centurionTitleFR string
	if err := db.QueryRow(
		`SELECT title_fr FROM milestone_catalog WHERE id = 'halo_infinite.matches.100'`).Scan(&centurionTitleFR); err != nil {
		t.Fatalf("centurion lookup: %v", err)
	}
	if centurionTitleFR == "" {
		t.Errorf("title_fr Centurion vide")
	}

	// A9 : les conditions localisées sont seedées pour les jalons qui en ont une.
	var condFR, condEN sql.NullString
	if err := db.QueryRow(
		`SELECT condition_fr, condition_en FROM milestone_catalog
		 WHERE id = 'halo_infinite.accuracy_consistent.30d'`).Scan(&condFR, &condEN); err != nil {
		t.Fatalf("condition lookup: %v", err)
	}
	if !condFR.Valid || condFR.String == "" {
		t.Errorf("condition_fr vide pour accuracy_consistent.30d (A9)")
	}
	if !condEN.Valid || condEN.String == "" {
		t.Errorf("condition_en vide pour accuracy_consistent.30d (A9)")
	}
}

// TestSeedMilestonesFromTOML_Idempotent : 2 runs = même count.
func TestSeedMilestonesFromTOML_Idempotent(t *testing.T) {
	db := openMetadataDB(t)

	path := filepath.Join(configTitlesRootHelper(t), "halo_infinite", "milestones", "catalog.toml")
	if err := seedMilestonesFromTOML(db, path); err != nil {
		t.Fatalf("run 1: %v", err)
	}
	var n1 int
	db.QueryRow(`SELECT COUNT(*) FROM milestone_catalog`).Scan(&n1)

	if err := seedMilestonesFromTOML(db, path); err != nil {
		t.Fatalf("run 2 (doit être no-op via ON CONFLICT): %v", err)
	}
	var n2 int
	db.QueryRow(`SELECT COUNT(*) FROM milestone_catalog`).Scan(&n2)

	if n1 != n2 {
		t.Fatalf("idempotence cassée : run1=%d run2=%d", n1, n2)
	}
}

// TestSeedMilestonesAllTitles_MultiTitleIteration : itère sur
// config/titles/*/milestones/catalog.toml.
func TestSeedMilestonesAllTitles_MultiTitleIteration(t *testing.T) {
	db := openMetadataDB(t)

	if err := seedMilestonesAllTitles(db, configTitlesRootHelper(t)); err != nil {
		t.Fatalf("seedMilestonesAllTitles: %v", err)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM milestone_catalog`).Scan(&count)
	if count == 0 {
		t.Fatalf("expected >0 milestones après seed all titles, got 0")
	}
}

// TestSeedMilestonesAllTitles_MissingRoot_NoOp : configTitlesRoot inexistant → no-op.
func TestSeedMilestonesAllTitles_MissingRoot_NoOp(t *testing.T) {
	db := openMetadataDB(t)

	tmp := filepath.Join(t.TempDir(), "definitely-not-existing")
	if err := seedMilestonesAllTitles(db, tmp); err != nil {
		t.Fatalf("expected no-op on missing root, got error: %v", err)
	}
}

// TestSeedMilestonesAllTitles_TitleWithoutCatalog_Skip : titre sans
// milestones/catalog.toml → skip silencieux.
func TestSeedMilestonesAllTitles_TitleWithoutCatalog_Skip(t *testing.T) {
	db := openMetadataDB(t)

	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "mythical_title"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := seedMilestonesAllTitles(db, tmp); err != nil {
		t.Fatalf("expected skip, got: %v", err)
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM milestone_catalog`).Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 rows (no catalog), got %d", count)
	}
}

// NB : pas de test d'idempotence de RegisterMilestonesSeedMigration ici. L'appeler
// enregistrerait seed_milestone_catalog_v1 dans le registre global PERSISTANT du
// binaire de test, ce que TestCanonicalCoversGlobalAndTitle (même package) verrait
// comme un step hors canonicalOrder (les seeds TOML dynamiques run-last via le
// fallback de canonicalRank, volontairement non listés — cf. seed_prestige_catalog_v1).
// La garde IsRegistered est le même wrapper trivial que prestige.go ; l'idempotence
// du seed lui-même est couverte par TestSeedMilestonesFromTOML_Idempotent.

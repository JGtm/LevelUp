//go:build integration

// Tests pour la migration seed du catalogue milestones (Phase 4 plan
// stabilisation 2026-05-22).
//
// Objectifs :
//   - seedMilestonesFromTOML peuple milestone_catalog avec les 13 entrées
//     d'Halo Infinite
//   - Idempotent : 2 runs successifs → même count
//   - Multi-titres : itère sur config/titles/*/milestones/catalog.toml
//   - TOML absent / parent absent : no-op gracieux
//   - RegisterMilestonesSeedMigration : idempotence du registre
package migration

import (
	"os"
	"path/filepath"
	"testing"
)

// configTitlesRootHelper retourne config/titles/ depuis apps/go-api/internal/migration.
func configTitlesRootHelper(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "config", "titles"))
	if err != nil {
		t.Fatalf("configTitlesRoot abs: %v", err)
	}
	return abs
}

// TestSeedMilestonesFromTOML_HaloInfinite_PopulatesRows : 13 milestones d'Halo
// Infinite chargés depuis le vrai TOML config.
func TestSeedMilestonesFromTOML_HaloInfinite_PopulatesRows(t *testing.T) {
	db := openMemDB(t)
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

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
	// Sanity check : le TOML doit avoir au moins quelques entrées centunion (matches.100).
	var centurionTitleFR string
	if err := db.QueryRow(
		`SELECT title_fr FROM milestone_catalog WHERE id = 'halo_infinite.matches.100'`).Scan(&centurionTitleFR); err != nil {
		t.Fatalf("centurion lookup: %v", err)
	}
	if centurionTitleFR == "" {
		t.Errorf("title_fr Centurion vide")
	}
}

// TestSeedMilestonesFromTOML_Idempotent : 2 runs = même count.
func TestSeedMilestonesFromTOML_Idempotent(t *testing.T) {
	db := openMemDB(t)
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

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

// TestSeedMilestonesAllTitles_MultiTitleIteration : la version multi-titres
// itère bien sur config/titles/*/milestones/catalog.toml.
func TestSeedMilestonesAllTitles_MultiTitleIteration(t *testing.T) {
	db := openMemDB(t)
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	if err := seedMilestonesAllTitles(db, configTitlesRootHelper(t)); err != nil {
		t.Fatalf("seedMilestonesAllTitles: %v", err)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM milestone_catalog`).Scan(&count)
	if count == 0 {
		t.Fatalf("expected >0 milestones après seed all titles, got 0")
	}
}

// TestSeedMilestonesAllTitles_MissingRoot_NoOp : si configTitlesRoot n'existe
// pas, no-op gracieux (cas tests :memory:).
func TestSeedMilestonesAllTitles_MissingRoot_NoOp(t *testing.T) {
	db := openMemDB(t)
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	tmp := filepath.Join(t.TempDir(), "definitely-not-existing")
	if err := seedMilestonesAllTitles(db, tmp); err != nil {
		t.Fatalf("expected no-op on missing root, got error: %v", err)
	}
}

// TestSeedMilestonesAllTitles_TitleWithoutCatalog_Skip : titre présent mais
// sans milestones/catalog.toml → skip silencieux.
func TestSeedMilestonesAllTitles_TitleWithoutCatalog_Skip(t *testing.T) {
	db := openMemDB(t)
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "mythical_title"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Pas de milestones/catalog.toml créé → la migration doit skip silencieux.
	if err := seedMilestonesAllTitles(db, tmp); err != nil {
		t.Fatalf("expected skip, got: %v", err)
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM milestone_catalog`).Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 rows (no catalog), got %d", count)
	}
}

// TestRegisterMilestonesSeedMigration_Idempotent : 2 appels successifs au
// même nom ne dupliquent pas la migration dans le registre.
func TestRegisterMilestonesSeedMigration_Idempotent(t *testing.T) {
	// Capturer le count avant.
	before := len(registry)
	RegisterMilestonesSeedMigration("/tmp/dummy")
	mid := len(registry)
	RegisterMilestonesSeedMigration("/tmp/dummy")
	after := len(registry)

	// Soit ajouté 1× (si pas déjà dans le registre), soit 0× (si init() a déjà couru).
	// Ce qui compte : after == mid.
	if after != mid {
		t.Errorf("expected no double-registration, before=%d mid=%d after=%d", before, mid, after)
	}
}

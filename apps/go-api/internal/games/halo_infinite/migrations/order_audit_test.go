package migrations

// order_audit_test.go — garde-fou de complétude d'ordre vu de l'union
// (registre global legacy + steps title-owned). Vit ici car ce package peut
// importer migration ET ses propres Steps() ; le test côté package migration ne
// voit que le registre global (Phase 1.5.1 B).
//
// Bout-en-bout : prouve aussi que les steps title-owned passent réellement par
// le runner (provider → combineSteps → RunSteps) et appliquent leur DDL.

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// TestCanonicalCoversGlobalAndTitle : chaque step (global + title) est dans
// migration.canonicalOrder, et canonicalOrder n'a pas d'entrée morte vis-à-vis
// de l'union. Déplacer un step package migration → title ne doit jamais sortir
// son Name de canonicalOrder ni en oublier un nouveau.
func TestCanonicalCoversGlobalAndTitle(t *testing.T) {
	order := migration.CanonicalOrder()
	inOrder := make(map[string]bool, len(order))
	for _, n := range order {
		inOrder[n] = true
	}

	present := make(map[string]bool)
	check := func(name string) {
		present[name] = true
		if !inOrder[name] {
			t.Errorf("step %q absent de migration.canonicalOrder (order.go) — l'ajouter à la bonne position", name)
		}
	}
	for _, m := range migration.All() { // legacy (registre global)
		check(m.Name)
	}
	for _, m := range Steps() { // title-owned
		check(m.Name)
	}
	for _, n := range order { // reverse : pas d'entrée morte
		if !present[n] {
			t.Errorf("canonicalOrder référence %q absent de (global + title) — entrée morte", n)
		}
	}
}

// TestTitleStepsRunEndToEnd : le step title-owned add_pve_schema, fourni via le
// provider, est bien exécuté par RunForDB et crée sa table (preuve bout-en-bout
// de la voie B).
func TestTitleStepsRunEndToEnd(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, migration.TargetSharedPvE); err != nil {
		t.Fatalf("RunForDB(SharedPvE): %v", err)
	}

	var n int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'pve_match_stats'",
	).Scan(&n); err != nil {
		t.Fatalf("query table: %v", err)
	}
	if n != 1 {
		t.Errorf("table pve_match_stats absente après migration title-owned (voie B cassée)")
	}
}

// TestTitleStepsRunEndToEnd_Metadata : la famille prestige metadata title-owned
// (create_prestige_metadata_schema + ALTER challenge_template), fournie via le
// provider, est exécutée par RunForDB(TargetMetadata) et crée ses tables.
// Restaure la couverture du test skip-guardé côté package migration (b10).
func TestTitleStepsRunEndToEnd_Metadata(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForDB(Metadata): %v", err)
	}

	for _, table := range []string{"challenge_template", "preset_arc", "preset_arc_step"} {
		var n int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", table,
		).Scan(&n); err != nil {
			t.Fatalf("query table %s: %v", table, err)
		}
		if n != 1 {
			t.Errorf("table %s absente après migration metadata title-owned (voie B cassée)", table)
		}
	}

	// Sanity : colonnes ajoutées par les ALTER title-owned (source + tagging).
	for _, col := range []string{"source", "lusr_components", "radar_axes", "is_long_term"} {
		var n int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'challenge_template' AND column_name = ?", col,
		).Scan(&n); err != nil {
			t.Fatalf("query column %s: %v", col, err)
		}
		if n != 1 {
			t.Errorf("challenge_template.%s absente — ALTER title-owned non appliqué", col)
		}
	}
}

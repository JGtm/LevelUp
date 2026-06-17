//go:build integration

package migration

// title_ledger_test.go — PMT-9 : preuve du seam title-aware au niveau du runner.
//   - le chemin par défaut (RunForDB) trace title_slug='halo_infinite' + écrit
//     title_schema_version (oracle a, mécanique du ledger) ;
//   - un TitleMigrationSet enregistré route SES steps et JAMAIS le registre global
//     (isolation, mécanique d'oracle b) avec une version distincte.

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestRunForDB_RecordsTitleSlugAndVersion : le défaut Halo écrit title_slug sur
// schema_migrations + une ligne title_schema_version (version = len canonicalOrder).
func TestRunForDB_RecordsTitleSlugAndVersion(t *testing.T) {
	db := openMemDB(t)

	target := TargetDB("test_ledger_target")
	Register(Migration{
		Name:        "test_ledger_step",
		TargetDB:    target,
		Description: "step de test pour le ledger PMT-9",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.Exec("CREATE TABLE IF NOT EXISTS test_ledger_marker (id INTEGER)")
			return err
		},
	})

	if err := RunForDB(db, target); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	var slug string
	if err := db.QueryRow(
		"SELECT title_slug FROM schema_migrations WHERE name = 'test_ledger_step'",
	).Scan(&slug); err != nil {
		t.Fatalf("read title_slug: %v", err)
	}
	if slug != DefaultSlug {
		t.Errorf("schema_migrations.title_slug = %q, want %q", slug, DefaultSlug)
	}

	var version int
	if err := db.QueryRow(
		"SELECT version FROM title_schema_version WHERE title_slug = ? AND target = ?",
		DefaultSlug, string(target),
	).Scan(&version); err != nil {
		t.Fatalf("read title_schema_version: %v", err)
	}
	if want := len(canonicalOrder); version != want {
		t.Errorf("title_schema_version.version = %d, want %d (len canonicalOrder)", version, want)
	}
}

// TestRunForTitleDB_RoutesRegisteredSetInIsolation : un set enregistré applique
// SES steps, JAMAIS le registre global (même target), avec son slug + sa version.
func TestRunForTitleDB_RoutesRegisteredSetInIsolation(t *testing.T) {
	db := openMemDB(t)

	const isoSlug = "test_iso_title"
	isoTarget := TargetDB("test_iso_target")

	// Step GLOBAL sur la même target : ne doit PAS s'appliquer via le set.
	Register(Migration{
		Name:        "test_global_only_step",
		TargetDB:    isoTarget,
		Description: "step global — ne doit pas fuiter dans un titre enregistré",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.Exec("CREATE TABLE IF NOT EXISTS test_global_marker (id INTEGER)")
			return err
		},
	})

	// Set du titre : son propre step + son propre ordre, zéro registre global.
	RegisterMigrationSet(TitleMigrationSet{
		Slug:           isoSlug,
		CanonicalOrder: []string{"test_set_step"},
		Steps: func(target TargetDB) []Migration {
			if target != isoTarget {
				return nil
			}
			return []Migration{{
				Name:        "test_set_step",
				TargetDB:    isoTarget,
				Description: "step appartenant au set du titre de test",
				ApplySchema: func(db *sql.DB) error {
					_, err := db.Exec("CREATE TABLE IF NOT EXISTS test_set_marker (id INTEGER)")
					return err
				},
			}}
		},
	})

	if err := RunForTitleDB(db, isoSlug, isoTarget); err != nil {
		t.Fatalf("RunForTitleDB(%s): %v", isoSlug, err)
	}

	if !tableExists2(t, db, "test_set_marker") {
		t.Errorf("test_set_marker absente — le step du set n'a pas été appliqué")
	}
	if tableExists2(t, db, "test_global_marker") {
		t.Errorf("test_global_marker présente — fuite du registre global dans un titre enregistré (isolation cassée)")
	}

	var slug string
	if err := db.QueryRow(
		"SELECT title_slug FROM schema_migrations WHERE name = 'test_set_step'",
	).Scan(&slug); err != nil {
		t.Fatalf("read title_slug: %v", err)
	}
	if slug != isoSlug {
		t.Errorf("schema_migrations.title_slug = %q, want %q", slug, isoSlug)
	}

	var version int
	if err := db.QueryRow(
		"SELECT version FROM title_schema_version WHERE title_slug = ? AND target = ?",
		isoSlug, string(isoTarget),
	).Scan(&version); err != nil {
		t.Fatalf("read title_schema_version: %v", err)
	}
	if version != 1 {
		t.Errorf("title_schema_version.version = %d, want 1 (len de l'ordre du set)", version)
	}
}

func tableExists2(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", name,
	).Scan(&n); err != nil {
		t.Fatalf("tableExists(%s): %v", name, err)
	}
	return n == 1
}

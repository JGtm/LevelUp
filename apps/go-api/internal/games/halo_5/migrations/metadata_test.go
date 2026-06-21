package migrations

// metadata_test.go — ROOT FIX assets Halo 5 : prouve l'ISOLATION metadata + le
// maintien de l'HÉRITAGE shared.
//
//   (a) metadata h5 → SES tables référentielles propres, VIDES, et AUCUNE
//       pollution des référentiels Halo Infinite (career_rank_translations,
//       citation_mappings, prestige, battlepass, playlists_catalog, …).
//   (b) shared h5 → hérite du schéma uniforme HINF (OwnsTarget ne couvre QUE
//       metadata → fallback complet pour shared).

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halo5 "levelup/go-api/internal/games/halo_5"
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

func rowCount(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("rowCount(%s): %v", table, err)
	}
	return n
}

// TestHalo5Metadata_IsolatedFromInfinite : la metadata h5 a SES tables (vides) et
// AUCUN référentiel HINF.
func TestHalo5Metadata_IsolatedFromInfinite(t *testing.T) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	Register()

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := migration.RunForTitleDB(db, halo5.TitleSlug, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForTitleDB(%s, metadata): %v", halo5.TitleSlug, err)
	}

	// (a) Les tables référentielles h5 existent.
	for _, tbl := range []string{
		"asset_translations", "medal_translations", "medal_definitions",
		"weapon_labels", "maps_catalog", "map_images_registry",
	} {
		if !tableExists(t, db, tbl) {
			t.Errorf("table h5 %q absente — set metadata h5 non appliqué", tbl)
		}
	}

	// Et elles sont VIDES (zéro seed — les fetchers CMS h5 les peupleront).
	for _, tbl := range []string{"medal_definitions", "weapon_labels", "maps_catalog"} {
		if n := rowCount(t, db, tbl); n != 0 {
			t.Errorf("%s contient %d lignes — un seed HINF a fuité (attendu 0)", tbl, n)
		}
	}

	// (b) AUCUN référentiel Halo Infinite (pollution) dans la metadata h5.
	for _, polluant := range []string{
		"career_rank_translations", // échelle 272 rangs HINF ≠ SR h5
		"citation_mappings",        // citations HINF
		"challenge_template",       // prestige HINF
		"preset_arc",               // prestige HINF
		"battlepass_track_definitions",
		"xbox_achievement_definitions",
		"playlists_catalog", // catalogue playlists HINF
		"csr_placement_thresholds",
		"mode_name_tr", // modes HINF
	} {
		if tableExists(t, db, polluant) {
			t.Errorf("table HINF %q présente dans la metadata h5 — POLLUTION (isolation cassée)", polluant)
		}
	}
}

// TestHalo5Shared_InheritsInfiniteSchema : le target shared (NON possédé par le
// set h5) retombe sur le fallback HINF complet → schéma uniforme.
func TestHalo5Shared_InheritsInfiniteSchema(t *testing.T) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	Register()

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := migration.RunForTitleDB(db, halo5.TitleSlug, migration.TargetShared); err != nil {
		t.Fatalf("RunForTitleDB(%s, shared): %v", halo5.TitleSlug, err)
	}

	for _, tbl := range []string{
		"match_registry", "match_participants", "medals_earned",
		"killer_victim_pairs", "xuid_aliases",
	} {
		if !tableExists(t, db, tbl) {
			t.Errorf("table HINF %q absente du shared h5 — héritage uniforme cassé", tbl)
		}
	}
}

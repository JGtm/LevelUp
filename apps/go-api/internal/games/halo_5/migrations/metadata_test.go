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
	"path/filepath"
	"runtime"
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
		// Référentiel succès Xbox h5 : brique metadata commune (même forme que HINF,
		// + colonne title_id). Possédée par le set h5 désormais (C4 title-agnostic).
		"xbox_achievement_definitions",
		// Référentiel milestones (Progression V2) : schéma propre au set h5 (le seed
		// reste vide ici, racine non injectée en test → no-op gracieux).
		"milestone_catalog",
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
		"playlists_catalog", // catalogue playlists HINF
		"csr_placement_thresholds",
		"mode_name_tr", // modes HINF
	} {
		if tableExists(t, db, polluant) {
			t.Errorf("table HINF %q présente dans la metadata h5 — POLLUTION (isolation cassée)", polluant)
		}
	}
}

// TestHalo5Metadata_WeaponRegistrySeeded : le set metadata h5 crée le registre
// d'armes CROSS-TITRE (weapons/weapon_ids/weapon_families) et le seede avec les
// armes Halo 5 + leurs stock_ids et RÔLES. Sans lui, resolveWeaponMeta retombe
// sur weapon_labels seul (aucun rôle) → aucune classe/rôle d'arme dans la
// FragDistribution (sunburst réduit aux classes API + résidu) sur H5 (bug B2b).
// Le registre est cross-titre par
// conception (PK title_slug+weapon_key, lectures title-scopées) : la présence de
// lignes halo_infinite dans la même table n'est PAS une pollution (≠ des
// référentiels title-specific isolés type career_ranks/playlists).
func TestHalo5Metadata_WeaponRegistrySeeded(t *testing.T) {
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

	// Les 3 tables du registre existent.
	for _, tbl := range []string{"weapons", "weapon_ids", "weapon_families"} {
		if !tableExists(t, db, tbl) {
			t.Errorf("table registre %q absente — h5_add_weapon_registry non appliqué", tbl)
		}
	}

	// Des armes Halo 5 avec un RÔLE non vide sont seedées (sans elles, aucun rôle
	// ne résout → donut vide).
	var h5WithRole int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM weapons WHERE title_slug = ? AND role IS NOT NULL AND role <> ''`,
		halo5.TitleSlug,
	).Scan(&h5WithRole); err != nil {
		t.Fatalf("count weapons halo_5 avec rôle: %v", err)
	}
	if h5WithRole == 0 {
		t.Error("aucune arme halo_5 avec rôle dans le registre — le donut « Frags par type d'arme » resterait vide")
	}

	// Des stock_ids Halo 5 sont mappés (résolution effective_weapon_id → weapon_key).
	var h5Ids int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM weapon_ids WHERE title_slug = ?`, halo5.TitleSlug,
	).Scan(&h5Ids); err != nil {
		t.Fatalf("count weapon_ids halo_5: %v", err)
	}
	if h5Ids == 0 {
		t.Error("aucun weapon_ids halo_5 — les kills H5 ne peuvent pas résoudre de rôle")
	}

	// Résolution de bout en bout : un stock_id H5 réel (Assault Rifle = 313138863,
	// cf. weaponRegistryH5Stock) doit joindre vers un rôle non vide.
	var role string
	if err := db.QueryRow(`
		SELECT COALESCE(w.role, '')
		FROM weapon_ids wi
		JOIN weapons w ON w.title_slug = wi.title_slug AND w.weapon_key = wi.weapon_key
		WHERE wi.title_slug = ? AND wi.id_value = '313138863'`, halo5.TitleSlug,
	).Scan(&role); err != nil {
		t.Fatalf("résolution stock_id 313138863: %v", err)
	}
	if role == "" {
		t.Error("stock_id H5 313138863 (Assault Rifle) ne résout aucun rôle")
	}
}

// TestHalo5Metadata_MilestoneCatalogSeeded : avec la racine config/titles injectée
// (SetMilestonesSeedRoot), le set metadata h5 SEED milestone_catalog depuis
// config/titles/halo_5/milestones/catalog.toml (C5). Sans ce seed, la couche
// Progression V2 ne peut servir aucun milestone pour Halo 5 (catalogue vide).
func TestHalo5Metadata_MilestoneCatalogSeeded(t *testing.T) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	Register()

	// Racine config/titles/ du repo (6 niveaux au-dessus du dossier du package).
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller indisponible")
	}
	pkgDir := filepath.Dir(thisFile)
	configTitles := filepath.Join(pkgDir, "..", "..", "..", "..", "..", "..", "config", "titles")
	SetMilestonesSeedRoot(configTitles)
	t.Cleanup(func() { SetMilestonesSeedRoot("") }) // ne pas fuiter l'état entre tests

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := migration.RunForTitleDB(db, halo5.TitleSlug, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForTitleDB(%s, metadata): %v", halo5.TitleSlug, err)
	}

	// Le catalogue h5 doit être peuplé, et UNIQUEMENT avec des entrées title_slug=halo_5.
	total := rowCount(t, db, "milestone_catalog")
	if total == 0 {
		t.Fatal("milestone_catalog h5 VIDE après seed — config/titles/halo_5/milestones/catalog.toml non chargé")
	}
	var h5Rows int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM milestone_catalog WHERE title_slug = ?`, halo5.TitleSlug,
	).Scan(&h5Rows); err != nil {
		t.Fatalf("count milestone_catalog halo_5: %v", err)
	}
	if h5Rows != total {
		t.Errorf("milestone_catalog contient %d lignes dont %d halo_5 — fuite cross-titre (attendu 100%% halo_5)", total, h5Rows)
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

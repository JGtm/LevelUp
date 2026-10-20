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
	"levelup/go-api/internal/games/weapons"
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

// TestHalo5Metadata_OrdreCanoniqueCouvreLesSteps — RATCHET. Le set h5 possède le
// target metadata : son CanonicalOrder est le SEUL ordre d'exécution appliqué
// (registry.runSteps). Un step absent de l'ordre part en fin de tri
// (sortByOrder : rang len(order)) — silencieusement, et donc potentiellement
// AVANT/APRÈS sa dépendance. Ce test exige la bijection stricte entre
// MetadataSteps() et metadataStepNames().
func TestHalo5Metadata_OrdreCanoniqueCouvreLesSteps(t *testing.T) {
	Register()

	order := metadataStepNames()
	inOrder := make(map[string]int, len(order))
	for i, n := range order {
		if _, dup := inOrder[n]; dup {
			t.Errorf("doublon dans metadataStepNames(): %q", n)
		}
		inOrder[n] = i
	}

	inSteps := make(map[string]bool, len(order))
	for _, m := range MetadataSteps() {
		inSteps[m.Name] = true
		if _, ok := inOrder[m.Name]; !ok {
			t.Errorf("step %q absent de metadataStepNames() — il serait trié en fin de cycle, hors de sa dépendance", m.Name)
		}
		if m.TargetDB != migration.TargetMetadata {
			t.Errorf("step %q du jeu metadata h5 vise le target %q", m.Name, m.TargetDB)
		}
	}
	for _, n := range order {
		if !inSteps[n] {
			t.Errorf("entrée morte dans metadataStepNames(): %q n'est fourni par aucun step", n)
		}
	}
}

// TestHalo5Metadata_PurgeWeaponFamiliesLabelsDansLeSet — RATCHET de l'incident du
// 2026-09-12 (provisioning halo_5 en échec à chaque boot). `weapon_families` est
// un référentiel CROSS-TITRE : la purge de ses colonnes de libellés doit être
// jouée sur la metadata h5 comme sur celle d'Infinite, faute de quoi le seed
// commun `weapons.ApplyRegistry` viole `name_en NOT NULL`. Elle doit en outre
// suivre le créateur de la table.
func TestHalo5Metadata_PurgeWeaponFamiliesLabelsDansLeSet(t *testing.T) {
	Register()

	order := metadataStepNames()
	posPurge, posRegistry := -1, -1
	for i, n := range order {
		switch n {
		case purgeWeaponFamiliesLabelsName:
			posPurge = i
		case "h5_add_weapon_registry":
			posRegistry = i
		}
	}
	if posPurge < 0 {
		t.Fatalf("%q absent de l'ordre canonique h5 — la metadata halo_5 garderait weapon_families.name_en NOT NULL", purgeWeaponFamiliesLabelsName)
	}
	if posRegistry < 0 {
		t.Fatal("h5_add_weapon_registry absent de l'ordre canonique h5")
	}
	if posPurge < posRegistry {
		t.Errorf("la purge (rang %d) précède h5_add_weapon_registry (rang %d) — elle doit suivre le créateur de weapon_families", posPurge, posRegistry)
	}

	// Le step fourni est CELUI du registre global (référence par nom, pas une copie).
	global, ok := migration.ByName(purgeWeaponFamiliesLabelsName)
	if !ok {
		t.Fatalf("%q introuvable dans le registre global", purgeWeaponFamiliesLabelsName)
	}
	var found *migration.Migration
	for _, m := range MetadataSteps() {
		if m.Name == purgeWeaponFamiliesLabelsName {
			cp := m
			found = &cp
		}
	}
	if found == nil {
		t.Fatalf("%q absent de MetadataSteps()", purgeWeaponFamiliesLabelsName)
	}
	if found.Description != global.Description || found.TargetDB != global.TargetDB {
		t.Errorf("le step h5 n'est pas celui du registre global (description/target divergents) — une copie a été introduite")
	}
}

// TestHalo5Metadata_PurgeLegacyWeaponFamilies — REPRODUCTION de l'incident, de
// bout en bout, sur une metadata h5 dans l'état PROD d'avant le 2026-09-08 :
// `weapon_families` porte encore name_en/name_fr NOT NULL et la purge n'est pas
// au ledger. Le cycle de migrations puis le seed cross-titre rejoué au boot
// (weapons.ApplyRegistry, via ReconcileRegistry) doivent tous deux réussir.
func TestHalo5Metadata_PurgeLegacyWeaponFamilies(t *testing.T) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	Register()

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// 1. Cycle nominal : le runner crée lui-même schema_migrations (aucune DDL de
	//    ledger recopiée dans ce test — une copie dériverait sans qu'on le voie).
	if err := migration.RunForTitleDB(db, halo5.TitleSlug, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForTitleDB initial: %v", err)
	}

	// 2. Retour à l'état LEGACY : table au schéma d'avant la purge (forme
	//    historique, cf. games/weapons/registry.go avant le 2026-09-08) et purge
	//    retirée du ledger.
	for _, stmt := range []string{
		`DROP TABLE weapon_families`,
		`CREATE TABLE weapon_families (
			family_key VARCHAR PRIMARY KEY,
			name_en    VARCHAR NOT NULL,
			name_fr    VARCHAR NOT NULL
		)`,
		`INSERT INTO weapon_families VALUES ('battle_rifle', 'Battle Rifle', 'Fusil de combat')`,
		`DELETE FROM schema_migrations WHERE name = '` + purgeWeaponFamiliesLabelsName + `'`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("mise en état legacy (%s): %v", stmt, err)
		}
	}

	// 3. Boot suivant : le cycle rejoue la purge...
	if err := migration.RunForTitleDB(db, halo5.TitleSlug, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForTitleDB sur metadata h5 legacy: %v", err)
	}
	var nameEnCols int
	if err := db.QueryRow(`SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'weapon_families' AND column_name IN ('name_en', 'name_fr')`).Scan(&nameEnCols); err != nil {
		t.Fatalf("inspection des colonnes: %v", err)
	}
	if nameEnCols != 0 {
		t.Errorf("weapon_families porte encore %d colonne(s) de libellé après la purge", nameEnCols)
	}
	// ... la ligne existante est conservée (rebuild CTAS-swap sans perte).
	if n := rowCount(t, db, "weapon_families"); n == 0 {
		t.Error("weapon_families vidée par la purge — perte de données")
	}

	// 4. Et le seed cross-titre rejoué à CHAQUE boot passe (c'était l'échec :
	//    « NOT NULL constraint failed: weapon_families.name_en »).
	if _, err := weapons.ReconcileRegistry(db, halo5.TitleSlug); err != nil {
		t.Fatalf("ReconcileRegistry après purge: %v", err)
	}
}

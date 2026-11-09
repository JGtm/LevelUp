//go:build integration

// Tests de metadata_weapon_names_to_translations_v1 (ADR 0040 D-5, plan i18n 8.4) : les noms d'armes
// stockés en colonnes appariées passent dans des tables par langue, sans perte, sans écraser une
// ligne existante, schéma (défauts, NOT NULL, clé primaire) conservé, idempotent, deux titres.
package migration

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ma5k : identifiant du film dont le bit 63 est posé (non bindable en paramètre).
const ma5kID = uint64(0xf5c335dfe7232c0f)

func openLegacyWeaponNamesDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	for _, s := range []string{
		`CREATE TABLE weapon_labels (weapon_id UBIGINT PRIMARY KEY, name_en VARCHAR NOT NULL, name_fr VARCHAR NOT NULL,
			icon_url VARCHAR DEFAULT 'https://cdn.example/none.png', weapon_type VARCHAR)`,
		`INSERT INTO weapon_labels (weapon_id, name_en, name_fr, weapon_type) VALUES
			(1, 'Melee', 'Corps à corps', 'melee'), (2, 'Vehicle', '', NULL), (3, 'Needler', 'Needler', 'power')`,
		fmt.Sprintf(`INSERT INTO weapon_labels (weapon_id, name_en, name_fr, icon_url) VALUES (%d, 'MA5K Avenger', 'MA5K Avenger', 'u')`, ma5kID),
		`CREATE TABLE weapon_label_names (weapon_id UBIGINT NOT NULL, lang VARCHAR NOT NULL, name VARCHAR NOT NULL,
			PRIMARY KEY (weapon_id, lang))`,
		`INSERT INTO weapon_label_names VALUES (3, 'en', 'Needler (API)')`,
		`CREATE TABLE weapon_name_labels (title_slug VARCHAR NOT NULL, weapon_key VARCHAR NOT NULL,
			name_en VARCHAR NOT NULL, name_fr VARCHAR NOT NULL, PRIMARY KEY (title_slug, weapon_key))`,
		`INSERT INTO weapon_name_labels VALUES
			('halo_infinite', 'hinf_br75', 'BR75', 'BR75'),
			('halo_infinite', 'hinf_sword', 'Energy Sword', 'Épée à énergie'),
			('halo_5', 'h5_light_rifle', 'Light Rifle', 'Fusil léger'),
			('halo_5', 'h5_ghost', 'Ghost', '')`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seed (%.70s): %v", s, err)
		}
	}
	return db
}

func weaponNameCounts(t *testing.T, db *sql.DB, table string) map[string]int {
	t.Helper()
	rows, err := db.Query(`SELECT lang, COUNT(*) FROM ` + table + ` GROUP BY lang`)
	if err != nil {
		t.Fatalf("counts %s: %v", table, err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var l string
		var n int
		if err := rows.Scan(&l, &n); err != nil {
			t.Fatal(err)
		}
		out[l] = n
	}
	return out
}

func TestWeaponNames_MigreSansPerteEtGardeLeSchema(t *testing.T) {
	db := openLegacyWeaponNamesDB(t)
	if err := applyWeaponNamesToTranslations(db); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// weapon_name_labels : une ligne par langue non vide, deux titres.
	counts := weaponNameCounts(t, db, "weapon_name_labels")
	if counts["en"] != 4 || counts["fr"] != 3 {
		t.Errorf("weapon_name_labels : %v, want en=4 fr=3 (fr vide de h5_ghost non copié)", counts)
	}
	for _, c := range []struct{ slug, key, lang, want string }{
		{"halo_infinite", "hinf_sword", "fr", "Épée à énergie"},
		{"halo_5", "h5_light_rifle", "fr", "Fusil léger"},
		{"halo_5", "h5_light_rifle", "en", "Light Rifle"},
		{"halo_5", "h5_ghost", "en", "Ghost"},
	} {
		if got := scalar[string](t, db, `SELECT name FROM weapon_name_labels WHERE title_slug = ? AND weapon_key = ? AND lang = ?`,
			c.slug, c.key, c.lang); got != c.want {
			t.Errorf("%s/%s/%s = %q, want %q", c.slug, c.key, c.lang, got, c.want)
		}
	}
	if n := scalar[int](t, db, `SELECT COUNT(*) FROM weapon_name_labels WHERE title_slug = 'halo_5' AND weapon_key = 'h5_ghost' AND lang = 'fr'`); n != 0 {
		t.Errorf("un nom vide ne doit pas devenir une ligne (%d)", n)
	}
	for _, col := range []string{"name_en", "name_fr"} {
		if ok, _ := columnExists(db, "weapon_name_labels", col); ok {
			t.Errorf("weapon_name_labels.%s devrait être retirée", col)
		}
	}

	// weapon_label_names : copie sans perte, la ligne existante n'est pas écrasée, bit 63 conservé.
	counts = weaponNameCounts(t, db, "weapon_label_names")
	if counts["en"] != 4 || counts["fr"] != 3 {
		t.Errorf("weapon_label_names : %v, want en=4 fr=3 (le fr vide de l'arme 2 n'est pas copié)", counts)
	}
	if got := scalar[string](t, db, `SELECT name FROM weapon_label_names WHERE weapon_id = 3 AND lang = 'en'`); got != "Needler (API)" {
		t.Errorf("la ligne existante a été écrasée : %q", got)
	}
	if got := scalar[string](t, db, `SELECT name FROM weapon_label_names WHERE weapon_id = 3 AND lang = 'fr'`); got != "Needler" {
		t.Errorf("arme 3 fr = %q", got)
	}
	if got := scalar[string](t, db, fmt.Sprintf(`SELECT name FROM weapon_label_names WHERE weapon_id = %d AND lang = 'en'`, ma5kID)); got != "MA5K Avenger" {
		t.Errorf("identifiant à bit 63 : %q", got)
	}

	// weapon_labels : colonnes de nom retirées, les autres, leurs valeurs et leurs défauts conservés.
	for _, col := range []string{"name_en", "name_fr"} {
		if ok, _ := columnExists(db, "weapon_labels", col); ok {
			t.Errorf("weapon_labels.%s devrait être retirée", col)
		}
	}
	if n := scalar[int](t, db, `SELECT COUNT(*) FROM weapon_labels`); n != 4 {
		t.Errorf("weapon_labels : %d lignes, want 4", n)
	}
	if got := scalar[string](t, db, `SELECT weapon_type FROM weapon_labels WHERE weapon_id = 3`); got != "power" {
		t.Errorf("weapon_type = %q", got)
	}
	if _, err := db.Exec(`INSERT INTO weapon_labels (weapon_id) VALUES (9)`); err != nil {
		t.Fatalf("insert après migration : %v", err)
	}
	if got := scalar[string](t, db, `SELECT icon_url FROM weapon_labels WHERE weapon_id = 9`); got != "https://cdn.example/none.png" {
		t.Errorf("DEFAULT d'icon_url perdu : %q", got)
	}

	// Clés primaires conservées / posées.
	if _, err := db.Exec(`INSERT INTO weapon_labels (weapon_id) VALUES (9)`); err == nil {
		t.Error("la clé primaire de weapon_labels devrait refuser un doublon")
	}
	if _, err := db.Exec(`INSERT INTO weapon_name_labels VALUES ('halo_5', 'h5_ghost', 'en', 'Ghost 2')`); err == nil {
		t.Error("la clé primaire (title_slug, weapon_key, lang) devrait refuser un doublon")
	}
	if _, err := db.Exec(`INSERT INTO weapon_label_names VALUES (3, 'en', 'x')`); err == nil {
		t.Error("la clé primaire (weapon_id, lang) devrait refuser un doublon")
	}
}

func TestWeaponNames_Idempotente(t *testing.T) {
	db := openLegacyWeaponNamesDB(t)
	if err := applyWeaponNamesToTranslations(db); err != nil {
		t.Fatalf("apply 1: %v", err)
	}
	before := []int{
		scalar[int](t, db, `SELECT COUNT(*) FROM weapon_name_labels`),
		scalar[int](t, db, `SELECT COUNT(*) FROM weapon_label_names`),
		scalar[int](t, db, `SELECT COUNT(*) FROM weapon_labels`),
	}
	if err := applyWeaponNamesToTranslations(db); err != nil {
		t.Fatalf("apply 2: %v", err)
	}
	after := []int{
		scalar[int](t, db, `SELECT COUNT(*) FROM weapon_name_labels`),
		scalar[int](t, db, `SELECT COUNT(*) FROM weapon_label_names`),
		scalar[int](t, db, `SELECT COUNT(*) FROM weapon_labels`),
	}
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("2e exécution : table %d = %d lignes au lieu de %d", i, after[i], before[i])
		}
	}
}

func TestWeaponNames_TablesAbsentesOuDejaMigrees(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := applyWeaponNamesToTranslations(db); err != nil {
		t.Fatalf("base vide : %v", err)
	}
	if err := MigrateWeaponNames(db); err != nil {
		t.Fatalf("base vide (export) : %v", err)
	}
	// Forme finale déjà en place : aucun effet, aucune donnée touchée.
	if err := ExecScript(db, `CREATE TABLE weapon_labels (weapon_id UBIGINT PRIMARY KEY, icon_url VARCHAR);
		INSERT INTO weapon_labels VALUES (1, 'u')`); err != nil {
		t.Fatal(err)
	}
	if err := ExecScript(db, WeaponTablesDDL()); err != nil {
		t.Fatalf("DDL des tables par langue : %v", err)
	}
	if err := ExecScript(db, WeaponTablesDDL()); err != nil {
		t.Fatalf("DDL des tables par langue, 2e exécution : %v", err)
	}
	if _, err := db.Exec(`INSERT INTO weapon_name_labels VALUES ('halo_5', 'k', 'fr', 'Nom')`); err != nil {
		t.Fatal(err)
	}
	if err := applyWeaponNamesToTranslations(db); err != nil {
		t.Fatalf("forme finale : %v", err)
	}
	if n := scalar[int](t, db, `SELECT COUNT(*) FROM weapon_name_labels`); n != 1 {
		t.Errorf("weapon_name_labels : %d lignes, want 1", n)
	}
	if n := scalar[int](t, db, `SELECT COUNT(*) FROM weapon_labels`); n != 1 {
		t.Errorf("weapon_labels : %d lignes, want 1", n)
	}
}

// Un registre rejoué sur une base déjà migrée par ailleurs (colonnes seules retirées de
// weapon_labels, weapon_name_labels déjà par langue) ne touche à rien.
func TestWeaponNames_UneSeuleTableALAncienneForme(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := ExecScript(db, WeaponTablesDDL()); err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{
		`INSERT INTO weapon_name_labels VALUES ('halo_5', 'k', 'en', 'Name')`,
		`CREATE TABLE weapon_labels (weapon_id UBIGINT PRIMARY KEY, name_en VARCHAR NOT NULL, name_fr VARCHAR NOT NULL)`,
		`INSERT INTO weapon_labels VALUES (5, 'Five', 'Cinq')`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	if err := applyWeaponNamesToTranslations(db); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if n := scalar[int](t, db, `SELECT COUNT(*) FROM weapon_name_labels`); n != 1 {
		t.Errorf("weapon_name_labels : %d lignes, want 1", n)
	}
	if got := scalar[string](t, db, `SELECT name FROM weapon_label_names WHERE weapon_id = 5 AND lang = 'fr'`); got != "Cinq" {
		t.Errorf("arme 5 fr = %q", got)
	}
}

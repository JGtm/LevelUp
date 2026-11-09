//go:build integration

// Tests de metadata_definition_texts_to_translations_v1 (ADR 0040 D-5, plan i18n 8.3) : les
// colonnes appariées _fr/_en passent dans les tables par langue, sans perte, sans écraser une
// ligne de traduction existante, schéma (défauts, NOT NULL, clé primaire) conservé, idempotent.
package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openDefinitionTextsDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	for _, s := range []string{
		`CREATE TABLE asset_translations (asset_id VARCHAR NOT NULL, asset_type VARCHAR NOT NULL, lang VARCHAR NOT NULL,
			name VARCHAR NOT NULL, description VARCHAR, fetched_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP), PRIMARY KEY (asset_id, asset_type, lang))`,
		`CREATE TABLE medal_translations (medal_name_id BIGINT NOT NULL, lang VARCHAR NOT NULL, name VARCHAR NOT NULL,
			description VARCHAR, PRIMARY KEY (medal_name_id, lang))`,
		`CREATE TABLE medal_definitions (medal_name_id BIGINT PRIMARY KEY, name_fr VARCHAR NOT NULL, name_en VARCHAR NOT NULL,
			description_fr VARCHAR DEFAULT '', description_en VARCHAR DEFAULT '', difficulty_index TINYINT DEFAULT 0, personal_score INTEGER DEFAULT 0)`,
		`CREATE TABLE commendation_definitions (commendation_id VARCHAR PRIMARY KEY, name_en VARCHAR NOT NULL, name_fr VARCHAR NOT NULL,
			description_en VARCHAR DEFAULT '', description_fr VARCHAR DEFAULT '', icon_url VARCHAR)`,
		`CREATE TABLE team_colors (team_id INTEGER PRIMARY KEY, name_en VARCHAR NOT NULL DEFAULT '', name_fr VARCHAR NOT NULL DEFAULT '', color VARCHAR)`,
		`CREATE TABLE xbox_achievement_definitions (achievement_id VARCHAR PRIMARY KEY, name_en VARCHAR NOT NULL DEFAULT '', name_fr VARCHAR NOT NULL DEFAULT '',
			description_en VARCHAR, description_fr VARCHAR, locked_desc_en VARCHAR, locked_desc_fr VARCHAR, gamerscore INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE milestone_catalog (id VARCHAR PRIMARY KEY, title_slug VARCHAR NOT NULL, title_en VARCHAR NOT NULL, title_fr VARCHAR NOT NULL,
			condition VARCHAR, condition_fr VARCHAR, condition_en VARCHAR)`,
		`INSERT INTO medal_definitions (medal_name_id, name_fr, name_en, description_fr, description_en) VALUES
			(1, 'Tueur', 'Killer', 'Un frag', 'One kill'), (2, 'Seul en FR', '', '', '')`,
		`INSERT INTO medal_translations VALUES (1, 'en-US', 'Killer (API)', 'API text')`,
		`INSERT INTO commendation_definitions (commendation_id, name_en, name_fr, description_en, description_fr) VALUES
			('c-1', 'Sharpshooter', 'Tireur d''elite', 'Hit things', 'Toucher des choses')`,
		`INSERT INTO team_colors (team_id, name_en, name_fr, color) VALUES (1, 'Red', 'Rouge', '#f00'), (2, 'Blue', 'Bleu', '#00f')`,
		`INSERT INTO xbox_achievement_definitions (achievement_id, name_en, name_fr, description_en, description_fr, locked_desc_en, locked_desc_fr, gamerscore)
			VALUES ('a-1', 'First', 'Premier', 'Do it', 'Fais-le', 'Locked', 'Verrouille', 10)`,
		`INSERT INTO milestone_catalog (id, title_slug, title_en, title_fr, condition, condition_fr, condition_en)
			VALUES ('m-1', 'halo_5', 'Centurion', 'Centurion', 'matches>=100', '100 matchs', '100 matches')`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seed (%.60s): %v", s, err)
		}
	}
	return db
}

func scalar[T any](t *testing.T, db *sql.DB, q string, args ...any) T {
	t.Helper()
	var v T
	if err := db.QueryRow(q, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return v
}

func TestDefinitionTexts_MigreSansPerteEtGardeLeSchema(t *testing.T) {
	db := openDefinitionTextsDB(t)
	if err := applyDefinitionTextsToTranslations(db); err != nil {
		t.Fatalf("apply: %v", err)
	}
	// Médailles : la ligne d'API existante n'est pas écrasée, le reste est copié.
	if got := scalar[string](t, db, `SELECT name FROM medal_translations WHERE medal_name_id = 1 AND lang = 'en-US'`); got != "Killer (API)" {
		t.Errorf("la traduction existante a été écrasée : %q", got)
	}
	if got := scalar[string](t, db, `SELECT name FROM medal_translations WHERE medal_name_id = 1 AND lang = 'fr-FR'`); got != "Tueur" {
		t.Errorf("medal fr = %q", got)
	}
	if got := scalar[string](t, db, `SELECT description FROM medal_translations WHERE medal_name_id = 1 AND lang = 'fr-FR'`); got != "Un frag" {
		t.Errorf("medal desc fr = %q", got)
	}
	if n := scalar[int](t, db, `SELECT COUNT(*) FROM medal_translations WHERE medal_name_id = 2`); n != 1 {
		t.Errorf("médaille à un seul texte : %d lignes, want 1 (fr seulement)", n)
	}
	// Types d'actifs.
	for _, c := range []struct {
		id, typ, lang, want string
	}{
		{"c-1", "commendation", "fr-FR", "Tireur d'elite"},
		{"1", "team_color", "en-US", "Red"},
		{"2", "team_color", "fr-FR", "Bleu"},
		{"a-1", "achievement", "fr-FR", "Premier"},
		{"a-1", "achievement_locked", "en-US", "Locked"},
		{"m-1", "milestone", "fr-FR", "Centurion"},
	} {
		if got := scalar[string](t, db, `SELECT name FROM asset_translations WHERE asset_id = ? AND asset_type = ? AND lang = ?`, c.id, c.typ, c.lang); got != c.want {
			t.Errorf("%s/%s/%s = %q, want %q", c.typ, c.id, c.lang, got, c.want)
		}
	}
	if got := scalar[string](t, db, `SELECT description FROM asset_translations WHERE asset_id = 'm-1' AND asset_type = 'milestone' AND lang = 'en-US'`); got != "100 matches" {
		t.Errorf("condition de jalon en = %q", got)
	}
	// Colonnes retirées, autres colonnes, valeurs et défauts conservés.
	for _, c := range []struct{ table, col string }{
		{"medal_definitions", "name_fr"}, {"medal_definitions", "description_en"}, {"commendation_definitions", "name_en"},
		{"team_colors", "name_fr"}, {"xbox_achievement_definitions", "locked_desc_fr"}, {"milestone_catalog", "title_en"},
		{"milestone_catalog", "condition_fr"},
	} {
		if ok, _ := columnExists(db, c.table, c.col); ok {
			t.Errorf("%s.%s devrait être retirée", c.table, c.col)
		}
	}
	if got := scalar[string](t, db, `SELECT condition FROM milestone_catalog WHERE id = 'm-1'`); got != "matches>=100" {
		t.Errorf("la formule technique `condition` a changé : %q", got)
	}
	if got := scalar[int](t, db, `SELECT gamerscore FROM xbox_achievement_definitions WHERE achievement_id = 'a-1'`); got != 10 {
		t.Errorf("gamerscore = %d", got)
	}
	// Défaut conservé : une insertion sans la colonne prend sa valeur par défaut.
	if _, err := db.Exec(`INSERT INTO medal_definitions (medal_name_id) VALUES (9)`); err != nil {
		t.Fatalf("insert après migration : %v", err)
	}
	if got := scalar[int](t, db, `SELECT difficulty_index FROM medal_definitions WHERE medal_name_id = 9`); got != 0 {
		t.Errorf("DEFAULT 0 perdu : %d", got)
	}
	// Clé primaire conservée.
	if _, err := db.Exec(`INSERT INTO medal_definitions (medal_name_id) VALUES (9)`); err == nil {
		t.Error("la clé primaire de medal_definitions devrait refuser un doublon")
	}
	if n := scalar[int](t, db, `SELECT COUNT(*) FROM team_colors`); n != 2 {
		t.Errorf("team_colors : %d lignes, want 2", n)
	}
}

func TestDefinitionTexts_Idempotente(t *testing.T) {
	db := openDefinitionTextsDB(t)
	if err := applyDefinitionTextsToTranslations(db); err != nil {
		t.Fatalf("apply 1: %v", err)
	}
	before := scalar[int](t, db, `SELECT COUNT(*) FROM asset_translations`) + scalar[int](t, db, `SELECT COUNT(*) FROM medal_translations`)
	if err := applyDefinitionTextsToTranslations(db); err != nil {
		t.Fatalf("apply 2: %v", err)
	}
	after := scalar[int](t, db, `SELECT COUNT(*) FROM asset_translations`) + scalar[int](t, db, `SELECT COUNT(*) FROM medal_translations`)
	if before != after {
		t.Errorf("2e exécution : %d lignes de traduction au lieu de %d", after, before)
	}
}

func TestDefinitionTexts_TablesAbsentesOuDejaSansColonnes(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := applyDefinitionTextsToTranslations(db); err != nil {
		t.Fatalf("base vide : %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE team_colors (team_id INTEGER PRIMARY KEY, color VARCHAR)`); err != nil {
		t.Fatal(err)
	}
	if err := applyDefinitionTextsToTranslations(db); err != nil {
		t.Fatalf("table sans colonnes appariées : %v", err)
	}
}

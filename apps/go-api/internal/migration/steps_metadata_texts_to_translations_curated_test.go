//go:build integration

// Tests de metadata_curated_texts_to_translations_v1 (ADR 0040 D-15, plan i18n 8.8.a) : les colonnes
// appariées de citation_mappings, challenge_template et preset_arc passent dans asset_translations
// sans perte, schéma conservé, idempotent.
package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openCuratedTextsDB(t *testing.T) *sql.DB {
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
		`CREATE TABLE citation_mappings (citation_name_norm VARCHAR PRIMARY KEY, citation_name_display VARCHAR NOT NULL,
			mapping_type VARCHAR NOT NULL DEFAULT 'medal', category VARCHAR, description VARCHAR, enabled BOOLEAN NOT NULL DEFAULT TRUE,
			citation_name_display_en VARCHAR, description_en VARCHAR)`,
		`CREATE TABLE challenge_template (id VARCHAR PRIMARY KEY, title_slug VARCHAR NOT NULL, label_en VARCHAR NOT NULL, label_fr VARCHAR NOT NULL,
			description_en VARCHAR, description_fr VARCHAR, normal_target DOUBLE NOT NULL, schema_version INTEGER NOT NULL DEFAULT 1)`,
		`CREATE TABLE preset_arc (id VARCHAR PRIMARY KEY, title_slug VARCHAR NOT NULL, title_en VARCHAR NOT NULL, title_fr VARCHAR NOT NULL,
			description_en VARCHAR, description_fr VARCHAR, schema_version INTEGER DEFAULT 1 NOT NULL)`,
		`INSERT INTO citation_mappings (citation_name_norm, citation_name_display, category, description, citation_name_display_en, description_en) VALUES
			('charge', 'A la charge', 'multiplayer', 'Prenez une base.', 'Storm the Walls', 'Take a base.'),
			('seul_fr', 'Seul en FR', 'misc', NULL, NULL, NULL)`,
		`INSERT INTO asset_translations (asset_id, asset_type, lang, name, description) VALUES ('charge', 'citation', 'en-US', 'Storm the Walls (API)', 'API text')`,
		`INSERT INTO challenge_template (id, title_slug, label_en, label_fr, description_en, description_fr, normal_target) VALUES
			('t-1', 'halo_infinite', 'Kills', 'Frags', 'Get kills', 'Faire des frags', 10), ('t-2', 'halo_infinite', 'Wins', 'Victoires', NULL, '', 3)`,
		`INSERT INTO preset_arc (id, title_slug, title_en, title_fr, description_en, description_fr) VALUES
			('a-1', 'halo_infinite', 'Warm-up', 'Echauffement', 'Start slow', 'Commencer doucement')`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seed (%.60s): %v", s, err)
		}
	}
	return db
}

func TestCuratedTexts_MigreSansPerteEtGardeLeSchema(t *testing.T) {
	db := openCuratedTextsDB(t)
	if err := applyCuratedTextsToTranslations(db); err != nil {
		t.Fatalf("apply: %v", err)
	}
	for _, c := range []struct{ id, typ, lang, want string }{
		{"charge", "citation", "fr-FR", "A la charge"},
		{"charge", "citation", "en-US", "Storm the Walls (API)"}, // la ligne existante n'est pas écrasée
		{"seul_fr", "citation", "fr-FR", "Seul en FR"},
		{"t-1", "challenge_template", "en-US", "Kills"},
		{"t-1", "challenge_template", "fr-FR", "Frags"},
		{"t-2", "challenge_template", "fr-FR", "Victoires"},
		{"a-1", "preset_arc", "fr-FR", "Echauffement"},
		{"a-1", "preset_arc", "en-US", "Warm-up"},
	} {
		if got := scalar[string](t, db, `SELECT name FROM asset_translations WHERE asset_id = ? AND asset_type = ? AND lang = ?`, c.id, c.typ, c.lang); got != c.want {
			t.Errorf("%s/%s/%s = %q, want %q", c.typ, c.id, c.lang, got, c.want)
		}
	}
	if got := scalar[string](t, db, `SELECT description FROM asset_translations WHERE asset_id = 'charge' AND asset_type = 'citation' AND lang = 'fr-FR'`); got != "Prenez une base." {
		t.Errorf("description fr de citation = %q", got)
	}
	if got := scalar[string](t, db, `SELECT description FROM asset_translations WHERE asset_id = 'charge' AND asset_type = 'citation' AND lang = 'en-US'`); got != "API text" {
		t.Errorf("description en de citation (ligne existante sans description : complétée ?) = %q", got)
	}
	if n := scalar[int](t, db, `SELECT COUNT(*) FROM asset_translations WHERE asset_id = 'seul_fr' AND asset_type = 'citation'`); n != 1 {
		t.Errorf("citation à un seul texte : %d lignes, want 1", n)
	}
	if n := scalar[int](t, db, `SELECT COUNT(*) FROM asset_translations WHERE asset_id = 't-2' AND asset_type = 'challenge_template'`); n != 2 {
		t.Errorf("modèle sans description : %d lignes, want 2 (nom en et fr)", n)
	}
	for _, c := range []struct{ table, col string }{
		{"citation_mappings", "citation_name_display"}, {"citation_mappings", "citation_name_display_en"},
		{"citation_mappings", "description"}, {"citation_mappings", "description_en"},
		{"challenge_template", "label_fr"}, {"challenge_template", "description_en"},
		{"preset_arc", "title_en"}, {"preset_arc", "description_fr"},
	} {
		if ok, _ := columnExists(db, c.table, c.col); ok {
			t.Errorf("%s.%s devrait être retirée", c.table, c.col)
		}
	}
	if got := scalar[string](t, db, `SELECT category FROM citation_mappings WHERE citation_name_norm = 'charge'`); got != "multiplayer" {
		t.Errorf("category = %q", got)
	}
	if got := scalar[float64](t, db, `SELECT normal_target FROM challenge_template WHERE id = 't-1'`); got != 10 {
		t.Errorf("normal_target = %v", got)
	}
	if _, err := db.Exec(`INSERT INTO citation_mappings (citation_name_norm) VALUES ('charge')`); err == nil {
		t.Error("la clé primaire de citation_mappings devrait refuser un doublon")
	}
	if _, err := db.Exec(`INSERT INTO preset_arc (id, title_slug) VALUES ('a-9', 'x')`); err != nil {
		t.Errorf("DEFAULT / NOT NULL de preset_arc perdus : %v", err)
	}
}

func TestCuratedTexts_Idempotente(t *testing.T) {
	db := openCuratedTextsDB(t)
	if err := applyCuratedTextsToTranslations(db); err != nil {
		t.Fatalf("apply 1: %v", err)
	}
	before := scalar[int](t, db, `SELECT COUNT(*) FROM asset_translations`)
	if err := applyCuratedTextsToTranslations(db); err != nil {
		t.Fatalf("apply 2: %v", err)
	}
	if after := scalar[int](t, db, `SELECT COUNT(*) FROM asset_translations`); after != before {
		t.Errorf("2e exécution : %d lignes au lieu de %d", after, before)
	}
}

func TestCuratedTexts_TablesAbsentes(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := applyCuratedTextsToTranslations(db); err != nil {
		t.Fatalf("base vide : %v", err)
	}
}

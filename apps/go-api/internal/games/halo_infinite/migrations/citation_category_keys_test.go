//go:build integration

// citation_category_keys_test.go — migration de données
// `normalize_citation_mappings_category_keys` (2026-08-04) : la colonne
// citation_mappings.category passe des libellés FR seedés aux clés canoniques.
//
// Tourne sur DuckDB :memory: (jamais sur data/) : le test crée le sous-ensemble de
// colonnes de citation_mappings qui l'intéresse, sans passer par la chaîne complète.
package migrations

import (
	"database/sql"
	"testing"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/migration"

	_ "github.com/duckdb/duckdb-go/v2"
)

// citationCategoryStep retrouve la migration par son nom dans Steps().
func citationCategoryStep(t *testing.T) migration.Migration {
	t.Helper()
	for _, m := range Steps() {
		if m.Name == "normalize_citation_mappings_category_keys" {
			return m
		}
	}
	t.Fatal("migration normalize_citation_mappings_category_keys absente de Steps()")
	return migration.Migration{}
}

func setupCitationCategoryTable(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
		CREATE TABLE citation_mappings (
			citation_name_norm VARCHAR PRIMARY KEY,
			category           VARCHAR
		);
		INSERT INTO citation_mappings VALUES
			('charge',        'Mode de jeu'),
			('splatter',      'Véhicule'),
			('headshot',      'Arme'),
			('teamwork',      'Multijoueur'),
			('company_pride', 'Spartan Companies'),
			('elite_slayer',  'Ennemi'),
			('already_keyed', 'game_mode'),
			('untouched',     NULL);
	`); err != nil {
		t.Fatalf("setup: %v", err)
	}
}

func categoryOf(t *testing.T, db *sql.DB, norm string) string {
	t.Helper()
	var cat sql.NullString
	if err := db.QueryRow(`SELECT category FROM citation_mappings WHERE citation_name_norm = ?`, norm).
		Scan(&cat); err != nil {
		t.Fatalf("lecture %s: %v", norm, err)
	}
	return cat.String
}

// TestNormalizeCitationCategoryKeys : conversion complète + idempotence (le second
// passage ne change rien) + non-régression des valeurs déjà canoniques et NULL.
func TestNormalizeCitationCategoryKeys(t *testing.T) {
	db := openEngMemDB(t)
	setupCitationCategoryTable(t, db)

	step := citationCategoryStep(t)
	for pass := 1; pass <= 2; pass++ {
		if err := step.ApplySchema(db); err != nil {
			t.Fatalf("passe %d: %v", pass, err)
		}
		want := map[string]string{
			"charge":        canonical.CommendationCategoryGameMode,
			"splatter":      canonical.CommendationCategoryVehicle,
			"headshot":      canonical.CommendationCategoryWeapon,
			"teamwork":      canonical.CommendationCategoryMultiplayer,
			"company_pride": canonical.CommendationCategorySpartanCompanies,
			"elite_slayer":  canonical.CommendationCategoryEnemy,
			"already_keyed": canonical.CommendationCategoryGameMode,
			"untouched":     "",
		}
		for norm, expected := range want {
			if got := categoryOf(t, db, norm); got != expected {
				t.Errorf("passe %d — %s.category = %q, want %q", pass, norm, got, expected)
			}
		}
	}
}

// TestNormalizeCitationCategoryKeys_NoHumanLabelLeft : après migration, aucune
// valeur ne survit hors du set canonique (le garde-rail qui permettra de retirer
// la tolérance aux libellés FR dans NormalizeCommendationCategory).
func TestNormalizeCitationCategoryKeys_NoHumanLabelLeft(t *testing.T) {
	db := openEngMemDB(t)
	setupCitationCategoryTable(t, db)
	if err := citationCategoryStep(t).ApplySchema(db); err != nil {
		t.Fatalf("apply: %v", err)
	}

	rows, err := db.Query(`SELECT DISTINCT category FROM citation_mappings WHERE category IS NOT NULL`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cat string
		if err := rows.Scan(&cat); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if canonical.NormalizeCommendationCategory(cat) != cat {
			t.Errorf("category %q reste un libellé humain après migration", cat)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
}

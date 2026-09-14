package migrations

// medal_clash_of_kings_test.go — la médaille VIP « Clash of Kings » (1053114074)
// est absente du catalogue officiel GameCMS : seule la migration
// `seed_clash_of_kings_medal` lui donne une identité. Ce test prouve les deux
// chemins : base neuve (INSERT) et base héritée portant la ligne bouchon
// « Unknown / Inconnue » (UPDATE), ainsi que l'idempotence.

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

const (
	clashOfKingsID       = 1053114074
	clashOfKingsNameEN   = "Clash of Kings"
	clashOfKingsDescEN   = "Kill a VIP while being a VIP yourself"
	clashOfKingsMedalTyp = "mode"
)

// TestClashOfKingsSeed_BaseNeuve : sur une base metadata vierge, la migration
// crée la ligne avec le nom anglais officieux (SpartanRecord + table de noms du
// film) et AUCUN libellé FR inventé.
func TestClashOfKingsSeed_BaseNeuve(t *testing.T) {
	db := runMetadataMigrations(t)
	assertClashOfKings(t, db)

	// Idempotence : rejouer le script ne duplique ni ne modifie la ligne.
	if err := migration.RunForDB(db, migration.TargetMetadata); err != nil {
		t.Fatalf("second RunForDB: %v", err)
	}
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM medal_definitions WHERE medal_name_id = ?`, clashOfKingsID,
	).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("lignes pour %d = %d, want 1", clashOfKingsID, n)
	}
}

// TestClashOfKingsSeed_LigneBouchon : une base héritée porte « Unknown /
// Inconnue » (ligne bouchon de l'ère Python, affichée telle quelle sur la page
// Médailles). La migration la corrige au lieu de la laisser en place.
func TestClashOfKingsSeed_LigneBouchon(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Schéma minimal + ligne bouchon posés AVANT les migrations.
	setup := []string{
		`CREATE TABLE medal_definitions (
			medal_name_id  BIGINT PRIMARY KEY,
			name_fr        VARCHAR,
			name_en        VARCHAR,
			description_fr VARCHAR,
			description_en VARCHAR,
			is_custom      BOOLEAN DEFAULT FALSE
		)`,
		`INSERT INTO medal_definitions
			(medal_name_id, name_fr, name_en, description_fr, description_en)
		 VALUES (1053114074, 'Inconnue', 'Unknown', 'Médaille inconnue.', 'Unknown medal.')`,
	}
	for _, q := range setup {
		if _, execErr := db.Exec(q); execErr != nil {
			t.Fatalf("setup %q: %v", q, execErr)
		}
	}

	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForDB(Metadata): %v", err)
	}
	assertClashOfKings(t, db)
}

func runMetadataMigrations(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForDB(Metadata): %v", err)
	}
	return db
}

func assertClashOfKings(t *testing.T, db *sql.DB) {
	t.Helper()
	var nameEN, descEN, medalType string
	var nameFR, descFR sql.NullString
	if err := db.QueryRow(`
		SELECT name_en, description_en, name_fr, description_fr, COALESCE(medal_type, '')
		FROM medal_definitions WHERE medal_name_id = ?`, clashOfKingsID,
	).Scan(&nameEN, &descEN, &nameFR, &descFR, &medalType); err != nil {
		t.Fatalf("médaille %d absente de medal_definitions: %v", clashOfKingsID, err)
	}
	if nameEN != clashOfKingsNameEN {
		t.Errorf("name_en = %q, want %q", nameEN, clashOfKingsNameEN)
	}
	if descEN != clashOfKingsDescEN {
		t.Errorf("description_en = %q, want %q", descEN, clashOfKingsDescEN)
	}
	if medalType != clashOfKingsMedalTyp {
		t.Errorf("medal_type = %q, want %q", medalType, clashOfKingsMedalTyp)
	}
	// Aucun libellé FR n'existe dans une source officielle : la colonne reste
	// vide pour que la chaîne COALESCE serve le nom anglais (exact) plutôt
	// qu'une traduction inventée ou le bouchon « Inconnue ».
	if nameFR.Valid && nameFR.String != "" {
		t.Errorf("name_fr = %q, want vide (aucune source FR officielle)", nameFR.String)
	}
	if descFR.Valid && descFR.String != "" {
		t.Errorf("description_fr = %q, want vide (aucune source FR officielle)", descFR.String)
	}
}

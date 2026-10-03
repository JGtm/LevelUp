package migrations

// medal_clash_of_kings_test.go — la médaille VIP « Clash of Kings » (1053114074)
// est absente du catalogue officiel GameCMS : seules les migrations
// `seed_clash_of_kings_medal` (base neuve) et `fix_clash_of_kings_placeholder`
// (base héritée) lui donnent une identité.
//
// Le cas héritant reproduit la base RÉELLE du poste : `medal_definitions` créée
// hors migrations à l'ère Python, donc SANS clé primaire (piège CLAUDE.md :
// `CREATE TABLE IF NOT EXISTS` n'ajoute jamais une PK à une table existante), et
// portant une ligne bouchon « Unknown / Inconnue ». C'est ce cas qui a échappé
// au premier correctif : `ON CONFLICT (medal_name_id)` y lève une erreur et
// l'étape entière n'applique rien.

import (
	"bytes"
	"database/sql"
	"os"
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

// TestClashOfKingsSeed_BaseNeuve : sur une base metadata vierge, les migrations
// créent la ligne avec le nom anglais sourcé (SpartanRecord + table de noms du
// film) et AUCUN libellé FR inventé. Rejouées, elles ne dupliquent rien.
func TestClashOfKingsSeed_BaseNeuve(t *testing.T) {
	db := ouvrirMemoire(t)
	jouerMetadata(t, db)
	assertClashOfKings(t, db)

	jouerMetadata(t, db)
	assertUneSeuleLigne(t, db)
}

// TestClashOfKingsSeed_BaseHeritee : table SANS clé primaire + ligne bouchon
// « Unknown / Inconnue » + traduction bouchon dans medal_translations (la chaîne
// FR de lecture la consulte AVANT medal_definitions). Après migrations, la page
// Médailles doit servir « Clash of Kings ».
func TestClashOfKingsSeed_BaseHeritee(t *testing.T) {
	db := ouvrirMemoire(t)
	for _, q := range []string{
		// Forme exacte de l'ère Python : aucune contrainte, aucun DEFAULT.
		`CREATE TABLE medal_definitions (
			medal_name_id  BIGINT,
			name_fr        VARCHAR,
			name_en        VARCHAR,
			description_fr VARCHAR,
			description_en VARCHAR,
			is_custom      BOOLEAN
		)`,
		`INSERT INTO medal_definitions
			(medal_name_id, name_fr, name_en, description_fr, description_en, is_custom)
		 VALUES (1053114074, 'Inconnue', 'Unknown', 'Médaille inconnue.', 'Unknown medal.', FALSE)`,
		`CREATE TABLE medal_translations (
			medal_name_id BIGINT NOT NULL,
			lang          VARCHAR NOT NULL,
			name          VARCHAR,
			description   VARCHAR,
			PRIMARY KEY (medal_name_id, lang)
		)`,
		`INSERT INTO medal_translations VALUES (1053114074, 'fr-FR', 'Inconnue', 'Médaille inconnue.')`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("setup %q: %v", q, err)
		}
	}

	jouerMetadata(t, db)
	assertClashOfKings(t, db)
	assertUneSeuleLigne(t, db)
	assertLabelServiFR(t, db)

	// Idempotence sur la base héritée aussi.
	jouerMetadata(t, db)
	assertClashOfKings(t, db)
	assertUneSeuleLigne(t, db)
}

// TestClashOfKingsSeed_BouchonAutreCasse : la valeur exacte du bouchon n'est pas
// connue (casse, variante). La réparation ne doit dépendre QUE de
// l'identifiant — c'est la condition sur `name_en = 'Unknown'` qui avait laissé
// passer le bug.
func TestClashOfKingsSeed_BouchonAutreCasse(t *testing.T) {
	db := ouvrirMemoire(t)
	for _, q := range []string{
		`CREATE TABLE medal_definitions (
			medal_name_id  BIGINT,
			name_fr        VARCHAR,
			name_en        VARCHAR,
			description_fr VARCHAR,
			description_en VARCHAR,
			is_custom      BOOLEAN
		)`,
		`INSERT INTO medal_definitions
			(medal_name_id, name_fr, name_en, description_fr, description_en, is_custom)
		 VALUES (1053114074, 'Inconnue', 'UNKNOWN MEDAL', 'Médaille inconnue.', '', FALSE)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("setup %q: %v", q, err)
		}
	}
	jouerMetadata(t, db)
	assertClashOfKings(t, db)
	assertLabelServiFR(t, db)
}

func ouvrirMemoire(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func jouerMetadata(t *testing.T, db *sql.DB) {
	t.Helper()
	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForDB(Metadata): %v", err)
	}
}

func assertUneSeuleLigne(t *testing.T, db *sql.DB) {
	t.Helper()
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

// assertLabelServiFR rejoue la chaîne de résolution FR de la page Médailles
// (platform/duckdb/medal_label_resolve.go, locale par défaut) : traduction
// locale, puis name_fr, puis traduction EN, puis name_en. C'est ce que l'API
// renvoie réellement — un test sur les seules colonnes ne l'aurait pas vu.
func assertLabelServiFR(t *testing.T, db *sql.DB) {
	t.Helper()
	var label string
	if err := db.QueryRow(`
		SELECT COALESCE(
			NULLIF(TRIM(mt_loc.name),''),
			NULLIF(TRIM(md.name_fr),''),
			NULLIF(TRIM(mt_en.name),''),
			NULLIF(TRIM(md.name_en),''),
			''
		)
		FROM medal_definitions md
		LEFT JOIN medal_translations mt_loc
		       ON mt_loc.medal_name_id = md.medal_name_id AND mt_loc.lang = 'fr-FR'
		LEFT JOIN medal_translations mt_en
		       ON mt_en.medal_name_id = md.medal_name_id AND mt_en.lang = 'en-US'
		WHERE md.medal_name_id = ?`, clashOfKingsID,
	).Scan(&label); err != nil {
		t.Fatalf("résolution du label FR: %v", err)
	}
	if label != clashOfKingsNameEN {
		t.Errorf("label servi en FR = %q, want %q", label, clashOfKingsNameEN)
	}
}

// TestMedalDefinitions_PasDOnConflict — ratchet. `medal_definitions` existe en
// base héritée SANS clé primaire : tout `ON CONFLICT (medal_name_id)` y lève une
// Binder Error et, le runner s'arrêtant au premier échec, bloque TOUTES les
// migrations metadata suivantes. C'est ce qui a rendu la médaille 1053114074
// « Inconnue » malgré son correctif. Le pattern autorisé est
// INSERT … WHERE NOT EXISTS.
func TestMedalDefinitions_PasDOnConflict(t *testing.T) {
	src, err := os.ReadFile("steps.go")
	if err != nil {
		t.Fatalf("lecture steps.go: %v", err)
	}
	if bytes.Contains(src, []byte("ON CONFLICT (medal_name_id)")) {
		t.Error("steps.go contient un `ON CONFLICT (medal_name_id)` : " +
			"medal_definitions n'a pas de clé primaire sur les bases héritées — " +
			"utiliser INSERT … WHERE NOT EXISTS")
	}
}

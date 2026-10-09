package migration

// steps_shared_recompute_mode_category_test.go - shared_recompute_mode_category_v1 sur une base
// synthétique (table, index, vues) et, si LEVELUP_MODECAT_REAL_DB est posée, sur une COPIE d'une
// base partagée réelle (jamais le fichier d'un serveur qui tourne).

import (
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis/modelabel"
)

const fixtureModeCategorySQL = `
CREATE TABLE match_registry (
	match_id VARCHAR PRIMARY KEY,
	start_time TIMESTAMP,
	pair_id VARCHAR,
	pair_name VARCHAR,
	map_id VARCHAR,
	mode_category VARCHAR,
	player_count SMALLINT DEFAULT 0
);
CREATE INDEX idx_registry_mode_category ON match_registry(mode_category);
CREATE INDEX idx_registry_map ON match_registry(map_id);
CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR);
CREATE VIEW v_match_full AS SELECT mr.* FROM match_registry mr;
CREATE VIEW mv_player_matches AS SELECT mr.match_id, mr.mode_category, mr.pair_name, mp.xuid
	FROM match_registry mr JOIN match_participants mp ON mp.match_id = mr.match_id;
CREATE VIEW v_sans_categorie AS SELECT match_id, pair_name FROM match_registry;
`

// lignes : match_id, pair_name, catégorie stockée (nil = NULL).
var fixtureModeCategoryRows = [][3]any{
	{"m1", "Ranked:Strongholds on Live Fire", "Other"},           // fausse (paire résolue après coup)
	{"m2", "Super Fiesta:Slayer on Catalyst - Forge", "Other"},   // fausse
	{"m3", "BTB:Fiesta Slayer on Highpower", "Fiesta"},           // fausse
	{"m4", "Super Husky Raid:CTF on Chasm", "Fiesta"},            // fausse
	{"m5", "Arena:Slayer on Bazaar", "Assassin"},                 // juste
	{"m6", "2d1a4b3c-5e6f-4a7b-8c9d-0e1f2a3b4c5d", "Other"},      // identifiant non résolu : juste
	{"m7", "Gruntpocalypse:Fiesta on Fathom Firefight", "Other"}, // fausse
	{"m8", "Quote'd:Mode", "BTB"},                                // fausse, apostrophe : littéral échappé
	{"m9", "Arena:Slayer on Bazaar", nil},                        // NULL : jamais touchée
	{"m10", "Arena:Slayer on Bazaar", ""},                        // vide : jamais touchée
	{"m11", nil, "Other"},                                        // pair_name NULL : jamais touchée
}

func openFixtureModeCategory(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := execScript(db, fixtureModeCategorySQL); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	for _, r := range fixtureModeCategoryRows {
		if _, err := db.Exec(`INSERT INTO match_registry (match_id, start_time, pair_name, map_id, mode_category)
			VALUES (?, TIMESTAMP '2026-10-01 10:00:00', ?, 'map1', ?)`, r[0], r[1], r[2]); err != nil {
			t.Fatalf("insert %v: %v", r[0], err)
		}
		if _, err := db.Exec(`INSERT INTO match_participants VALUES (?, 'x1')`, r[0]); err != nil {
			t.Fatalf("insert participant: %v", err)
		}
	}
	return db
}

func categoriesParMatch(t *testing.T, db *sql.DB) map[string]string {
	t.Helper()
	rows, err := db.Query(`SELECT match_id, COALESCE(mode_category, '<NULL>') FROM match_registry`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close() //nolint:errcheck // test
	out := map[string]string{}
	for rows.Next() {
		var id, cat string
		if err := rows.Scan(&id, &cat); err != nil {
			t.Fatal(err)
		}
		out[id] = cat
	}
	return out
}

func TestRecomputeModeCategory_CorrigeLesFaussesEtLaisseLeReste(t *testing.T) {
	db := openFixtureModeCategory(t)
	ctx := t.Context()
	ddl, err := ddlDeTable(ctx, db, registryTable)
	if err != nil {
		t.Fatal(err)
	}
	index, err := ddlDesIndex(ctx, db, registryTable)
	if err != nil {
		t.Fatal(err)
	}
	avant, err := fingerprintsHorsCategorie(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	vuesAvant := ddlDesVues(t, db)

	if err := applyRecomputeModeCategory(db); err != nil {
		t.Fatalf("migration: %v", err)
	}

	want := map[string]string{
		"m1": "Ranked", "m2": "Super Fiesta", "m3": "BTB", "m4": "Husky Raid", "m5": "Assassin",
		"m6": "Other", "m7": "Firefight", "m8": "Other", "m9": "<NULL>", "m10": "", "m11": "Other",
	}
	got := categoriesParMatch(t, db)
	for id, w := range want {
		if got[id] != w {
			t.Errorf("%s: catégorie %q, attendu %q", id, got[id], w)
		}
	}
	apres, err := fingerprintsHorsCategorie(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	for name, e := range avant {
		if apres[name] != e {
			t.Errorf("%s: empreinte hors catégorie changée %+v -> %+v", name, e, apres[name])
		}
	}
	if err := verifierSchemaIdentique(ctx, db, registryTable, ddl, keptIndexDDL(index)); err != nil {
		t.Errorf("schéma : %v", err)
	}
	if names, err := coveringIndexNames(ctx, db); err != nil || len(names) != 0 {
		t.Errorf("index de mode_category restants %v (err %v)", names, err)
	}
	if n := scanInt(t, db, `SELECT COUNT(*) FROM duckdb_indexes() WHERE table_name = 'match_registry'`); n != 1 {
		t.Errorf("index conservés = %d, attendu 1 (idx_registry_map)", n)
	}
	if vues := ddlDesVues(t, db); vues != vuesAvant {
		t.Errorf("DDL des vues changé :\navant %s\naprès %s", vuesAvant, vues)
	}
	if n := scanInt(t, db, `SELECT COUNT(*) FROM match_registry`); n != int64(len(fixtureModeCategoryRows)) {
		t.Errorf("lignes = %d, attendu %d", n, len(fixtureModeCategoryRows))
	}
}

func ddlDesVues(t *testing.T, db *sql.DB) string {
	t.Helper()
	rows, err := db.Query(`SELECT view_name || ':' || sql FROM duckdb_views() WHERE NOT internal ORDER BY view_name`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close() //nolint:errcheck // test
	var parts []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "\n")
}

func TestRecomputeModeCategory_Idempotente(t *testing.T) {
	db := openFixtureModeCategory(t)
	if err := applyRecomputeModeCategory(db); err != nil {
		t.Fatal(err)
	}
	oid := tableOID(t, db, registryTable)
	avant := categoriesParMatch(t, db)
	if err := applyRecomputeModeCategory(db); err != nil {
		t.Fatalf("second passage: %v", err)
	}
	if got := tableOID(t, db, registryTable); got != oid {
		t.Errorf("base déjà migrée reconstruite (oid %d -> %d) : aucune écriture attendue", oid, got)
	}
	for id, c := range categoriesParMatch(t, db) {
		if avant[id] != c {
			t.Errorf("%s: %q -> %q au second passage", id, avant[id], c)
		}
	}
}

func TestRecomputeModeCategory_BaseJusteIndexeeSeulementDesindexee(t *testing.T) {
	db := openFixtureModeCategory(t)
	// Rend toutes les lignes justes sans passer par la migration : base saine mais indexée.
	if err := applyRecomputeModeCategory(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE INDEX idx_registry_mode_category ON match_registry(mode_category)`); err != nil {
		t.Fatal(err)
	}
	oid := tableOID(t, db, registryTable)
	if err := applyRecomputeModeCategory(db); err != nil {
		t.Fatal(err)
	}
	if got := tableOID(t, db, registryTable); got != oid {
		t.Errorf("table reconstruite (oid %d -> %d) alors que seule l'index devait partir", oid, got)
	}
	if names, _ := coveringIndexNames(t.Context(), db); len(names) != 0 {
		t.Errorf("index restants %v", names)
	}
}

func TestRecomputeModeCategory_TitreSansCategorieNonTouche(t *testing.T) {
	db := openFixtureModeCategory(t)
	if _, err := db.Exec(`UPDATE match_registry SET mode_category = NULL`); err != nil {
		t.Fatal(err)
	}
	if err := applyRecomputeModeCategory(db); err != nil {
		t.Fatal(err)
	}
	for id, c := range categoriesParMatch(t, db) {
		if c != "<NULL>" {
			t.Errorf("%s: %q, une catégorie NULL ne se remplit pas", id, c)
		}
	}
}

// TestRecomputeModeCategory_MutationDeControle : la migration est jugée sur le résultat. Une
// règle inversée (catégorie stockée prise pour juste) laisserait des paires fausses : le garde
// de cardinalité/vérification ou la table d'attendus doit alors rougir.
func TestRecomputeModeCategory_ParesFaussesDetecteesParLeGarde(t *testing.T) {
	db := openFixtureModeCategory(t)
	pairs, err := readPairCategories(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}
	wrong, lines := wrongCategories(pairs)
	if lines != 6 || len(wrong) != 6 {
		t.Fatalf("paires fausses = %d (%d lignes), attendu 6 / 6 : %v", len(wrong), lines, wrong)
	}
	if err := verifierRecalcul2(t, db); err == nil {
		t.Error("la vérification doit refuser une base dont des paires sont encore fausses")
	}
}

// verifierRecalcul2 joue la vérification de fin de swap sur la base telle quelle (non migrée).
func verifierRecalcul2(t *testing.T, db *sql.DB) error {
	t.Helper()
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	ddl, err := ddlDeTable(ctx, tx, registryTable)
	if err != nil {
		t.Fatal(err)
	}
	index, err := ddlDesIndex(ctx, tx, registryTable)
	if err != nil {
		t.Fatal(err)
	}
	avant, err := fingerprintsHorsCategorie(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}
	return verifierRecalcul(ctx, tx, ddl, index, avant)
}

func TestCategoryCaseExpr_EchappeLesApostrophes(t *testing.T) {
	got := categoryCaseExpr(map[string]string{"a'b": "X", "a": "Y"})
	want := "CASE pair_name WHEN 'a' THEN 'Y' WHEN 'a''b' THEN 'X' ELSE mode_category END"
	if got != want {
		t.Errorf("expression %q, attendu %q", got, want)
	}
}

// TestRecomputeModeCategory_CopieReelle : même contrôle sur une COPIE d'une base partagée réelle.
// LEVELUP_MODECAT_REAL_DB = chemin d'une copie en lecture (le test en refait une copie).
func TestRecomputeModeCategory_CopieReelle(t *testing.T) {
	src := os.Getenv("LEVELUP_MODECAT_REAL_DB")
	if src == "" {
		t.Skip("LEVELUP_MODECAT_REAL_DB non posée : contrôle sur copie réelle non demandé")
	}
	dst := filepath.Join(t.TempDir(), "shared.duckdb")
	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("duckdb", dst)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() //nolint:errcheck // test
	ctx := t.Context()
	pairs, err := readPairCategories(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	wrong, lines := wrongCategories(pairs)
	t.Logf("AVANT : %d paires fausses, %d lignes fausses", len(wrong), lines)
	cross := map[string]int64{}
	for _, p := range pairs {
		if want := modelabel.InferCategory(p.pair); want != p.stored {
			cross[p.stored+" -> "+want] += p.rows
		}
	}
	t.Logf("AVANT, stockée -> canonique : %v", cross)
	avant, err := fingerprintsHorsCategorie(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	vuesAvant := ddlDesVues(t, db)
	rowsAvant := scanInt(t, db, `SELECT COUNT(*) FROM match_registry`)

	if err := applyRecomputeModeCategory(db); err != nil {
		t.Fatalf("migration: %v", err)
	}

	pairs, err = readPairCategories(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if w2, l2 := wrongCategories(pairs); l2 != 0 {
		t.Errorf("APRES : %d paires, %d lignes encore fausses", len(w2), l2)
	}
	apres, err := fingerprintsHorsCategorie(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	for name, e := range avant {
		if apres[name] != e {
			t.Errorf("%s: empreinte hors catégorie changée", name)
		}
	}
	if got := scanInt(t, db, `SELECT COUNT(*) FROM match_registry`); got != rowsAvant {
		t.Errorf("lignes %d -> %d", rowsAvant, got)
	}
	if vues := ddlDesVues(t, db); vues != vuesAvant {
		t.Error("DDL des vues changé")
	}
	if names, _ := coveringIndexNames(ctx, db); len(names) != 0 {
		t.Errorf("index de mode_category restants : %v", names)
	}
	oid := tableOID(t, db, registryTable)
	if err := applyRecomputeModeCategory(db); err != nil {
		t.Fatalf("second passage: %v", err)
	}
	if tableOID(t, db, registryTable) != oid {
		t.Error("second passage : table reconstruite")
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close() //nolint:errcheck // lecture
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

package duckdbbackup

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// exportDB crée une base source par ddl, l'exporte en Parquet sous staging/<key>/ et rend le
// dossier de staging.
func exportDB(t *testing.T, key string, ddl string) string {
	t.Helper()
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.duckdb")
	db, err := sql.Open("duckdb", srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("source: %v", err)
	}
	_ = db.Close()
	staging := filepath.Join(dir, "staging")
	if _, err := ExportTarget(context.Background(), Target{Key: key, Path: srcPath},
		filepath.Join(staging, key), 1); err != nil {
		t.Fatalf("export: %v", err)
	}
	return staging
}

func importTo(t *testing.T, staging, target string, finalize FinalizeFunc) *ImportResult {
	t.Helper()
	res, err := ImportFromStaging(context.Background(), staging, func(string) string { return target }, finalize, false)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	return res
}

func openRestored(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func countConstraints(t *testing.T, db *sql.DB, table, kind string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM duckdb_constraints()
		WHERE table_name = ? AND constraint_type = ?`, table, kind).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func scalar(t *testing.T, db *sql.DB, q string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q).Scan(&n); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return n
}

// epochDDL : base « d'époque » avec séquence, clés, défaut, index et deux vues dont l'une lit
// l'autre (dans l'ordre alphabétique inverse de leur dépendance).
const epochDDL = `
	CREATE SEQUENCE ev_seq START 1;
	CREATE TABLE schema_migrations (name VARCHAR PRIMARY KEY, applied_at TIMESTAMP);
	INSERT INTO schema_migrations VALUES ('step_a', '2026-01-01'), ('step_b', '2026-01-02');
	CREATE TABLE medals (match_id VARCHAR, xuid VARCHAR, n INTEGER DEFAULT 0, PRIMARY KEY (match_id, xuid));
	INSERT INTO medals VALUES ('m1', 'x1', 3), ('m1', 'x2', 1);
	CREATE TABLE aliases (xuid VARCHAR, gamertag VARCHAR, UNIQUE (gamertag));
	INSERT INTO aliases VALUES ('x1', 'A');
	CREATE TABLE events (id BIGINT PRIMARY KEY DEFAULT nextval('ev_seq'), v INTEGER);
	INSERT INTO events (v) VALUES (1), (2);
	CREATE INDEX idx_events_v ON events(v);
	CREATE VIEW b_events AS SELECT * FROM events;
	CREATE VIEW a_events_latest AS SELECT max(id) AS id FROM b_events;
`

// TestImport_RestaureLaBaseDansLEtatDeSonEpoque : export puis import → clés primaires et
// UNIQUE, défaut, séquence, index et vues de la base sauvegardée sont présents ; les lignes sont
// identiques ; les écritures de prod qui exigent une clé (ON CONFLICT, INSERT OR IGNORE)
// réussissent ; la finalisation est appelée.
func TestImport_RestaureLaBaseDansLEtatDeSonEpoque(t *testing.T) {
	staging := exportDB(t, "db1", epochDDL)
	target := filepath.Join(t.TempDir(), "out", "db1.duckdb")
	var finalized []string
	res := importTo(t, staging, target, func(_ context.Context, key string, _ *sql.DB) error {
		finalized = append(finalized, key)
		return nil
	})
	if len(res.Restored) != 1 || len(res.FallbackTables) != 0 || len(res.SchemaFailures) != 0 {
		t.Fatalf("résultat inattendu : %+v", res)
	}
	if len(finalized) != 1 {
		t.Fatalf("finalisation non appelée : %v", finalized)
	}
	db := openRestored(t, target)
	for _, c := range []struct{ table, kind string }{
		{"schema_migrations", "PRIMARY KEY"}, {"medals", "PRIMARY KEY"}, {"aliases", "UNIQUE"}, {"events", "PRIMARY KEY"},
	} {
		if countConstraints(t, db, c.table, c.kind) != 1 {
			t.Errorf("%s : contrainte %s absente après restauration", c.table, c.kind)
		}
	}
	if n := scalar(t, db, `SELECT count(*) FROM schema_migrations`); n != 2 {
		t.Errorf("registre restauré : %d lignes, want 2", n)
	}
	if _, err := db.Exec(`INSERT INTO medals (match_id, xuid, n) VALUES ('m1', 'x1', 9)
		ON CONFLICT (match_id, xuid) DO NOTHING`); err != nil {
		t.Fatalf("ON CONFLICT sur la base restaurée : %v", err)
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO aliases VALUES ('x9', 'A')`); err != nil {
		t.Fatalf("INSERT OR IGNORE sur la base restaurée : %v", err)
	}
	if _, err := db.Exec(`INSERT INTO medals (match_id, xuid) VALUES ('m2', 'x1')`); err != nil {
		t.Fatal(err)
	}
	if n := scalar(t, db, `SELECT n FROM medals WHERE match_id = 'm2'`); n != 0 {
		t.Errorf("défaut de la base sauvegardée absent : n=%d", n)
	}
	if _, err := db.Exec(`INSERT INTO events (v) VALUES (3)`); err != nil {
		t.Fatalf("séquence restaurée en retard (identifiant déjà pris) : %v", err)
	}
	if n := scalar(t, db, `SELECT id FROM a_events_latest`); n != 3 {
		t.Errorf("vues restaurées : max(id) = %d, want 3", n)
	}
	if n := scalar(t, db, `SELECT count(*) FROM duckdb_indexes() WHERE index_name = 'idx_events_v'`); n != 1 {
		t.Error("index secondaire non restauré")
	}
}

// TestImport_SansSchemaRepliSansClesNiPerte : sauvegarde antérieure à l'export du schéma →
// tables importées par CREATE TABLE AS, comptées ; aucune ligne écartée.
func TestImport_SansSchemaRepliSansClesNiPerte(t *testing.T) {
	staging := exportDB(t, "db1", `CREATE TABLE t (v INTEGER PRIMARY KEY); INSERT INTO t VALUES (1), (2);`)
	removeDDL(t, staging)
	target := filepath.Join(t.TempDir(), "o.duckdb")
	res := importTo(t, staging, target, nil)
	if len(res.FallbackTables) != 1 || res.FallbackTables[0] != "db1:t" {
		t.Fatalf("FallbackTables = %v", res.FallbackTables)
	}
	if n := scalar(t, openRestored(t, target), `SELECT count(*) FROM t`); n != 2 {
		t.Fatalf("lignes perdues : %d", n)
	}
}

// TestImport_BaseSansCleAvecDoublonsToutesLesLignes : une base sauvegardée qui avait perdu ses
// clés (doublons) est restaurée telle quelle — aucune ligne écartée à la restauration.
func TestImport_BaseSansCleAvecDoublonsToutesLesLignes(t *testing.T) {
	staging := exportDB(t, "db1", `CREATE TABLE s (name VARCHAR, ts TIMESTAMP);
		INSERT INTO s VALUES ('s', '2026-01-01'), ('s', '2026-03-01'), (NULL, NULL);`)
	target := filepath.Join(t.TempDir(), "db1.duckdb")
	res := importTo(t, staging, target, nil)
	if len(res.FallbackTables) != 0 {
		t.Fatalf("FallbackTables = %v", res.FallbackTables)
	}
	if n := scalar(t, openRestored(t, target), `SELECT count(*) FROM s`); n != 3 {
		t.Fatalf("lignes restaurées : %d, want 3", n)
	}
}

// TestImport_DernierExportSeulement : seul le dernier export d'une table fait foi.
func TestImport_DernierExportSeulement(t *testing.T) {
	staging := exportDB(t, "db1", `CREATE TABLE r (k VARCHAR PRIMARY KEY, v INTEGER); INSERT INTO r VALUES ('a', 1);`)
	old := filepath.Join(staging, "db1", "r_20000101_000000.parquet")
	db0, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db0.Exec(`COPY (SELECT 'zz' AS k, 0 AS v) TO '` + filepath.ToSlash(old) + `' (FORMAT PARQUET)`); err != nil {
		t.Fatal(err)
	}
	_ = db0.Close()
	target := filepath.Join(t.TempDir(), "db1.duckdb")
	importTo(t, staging, target, nil)
	if n := scalar(t, openRestored(t, target), `SELECT count(*) FROM r WHERE k = 'a'`); n != 1 {
		t.Fatal("dernier export non importé")
	}
	if n := scalar(t, openRestored(t, target), `SELECT count(*) FROM r`); n != 1 {
		t.Fatalf("un export ancien a été importé : %d lignes", n)
	}
}

// TestImport_EchecRemetLaBaseEnPlace : une finalisation en échec laisse l'ancienne base intacte.
func TestImport_EchecRemetLaBaseEnPlace(t *testing.T) {
	staging := exportDB(t, "db1", `CREATE TABLE t (v INTEGER); INSERT INTO t VALUES (1);`)
	target := filepath.Join(t.TempDir(), "db1.duckdb")
	if err := os.WriteFile(target, []byte("ancienne"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ImportFromStaging(context.Background(), staging, func(string) string { return target },
		func(context.Context, string, *sql.DB) error { return errors.New("boom") }, false)
	if err == nil {
		t.Fatal("une finalisation en échec doit faire échouer l'import")
	}
	got, readErr := os.ReadFile(target)
	if readErr != nil || string(got) != "ancienne" {
		t.Fatalf("ancienne base non remise en place : %q (err=%v)", got, readErr)
	}
}

// removeDDL retire le schéma exporté du staging (sauvegarde antérieure à son introduction).
func removeDDL(t *testing.T, staging string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(staging, "*", ddlFilePrefix+"*"))
	if err != nil || len(matches) == 0 {
		t.Fatalf("schéma introuvable dans le staging (err=%v)", err)
	}
	for _, m := range matches {
		if err := os.Remove(m); err != nil {
			t.Fatal(err)
		}
	}
}

// TestImport_SchemaTronqueDegradeLaCleSeulement : le `_ddl_` du cycle, tronqué (export
// interrompu, disque plein), ne fait pas échouer la restauration : la clé concernée retombe sur
// des tables sans clé (comptées), sans perte de ligne, et les autres clés sont restaurées avec
// leur schéma.
func TestImport_SchemaTronqueDegradeLaCleSeulement(t *testing.T) {
	staging := exportDB(t, "db1", `CREATE TABLE t (v INTEGER PRIMARY KEY); INSERT INTO t VALUES (1), (2);`)
	other := exportDB(t, "db2", `CREATE TABLE u (v INTEGER PRIMARY KEY); INSERT INTO u VALUES (1);`)
	if err := os.Rename(filepath.Join(other, "db2"), filepath.Join(staging, "db2")); err != nil {
		t.Fatal(err)
	}
	// Le schéma du MÊME cycle que les Parquet, tronqué.
	ddlFiles, err := filepath.Glob(filepath.Join(staging, "db1", ddlFilePrefix+"*"))
	if err != nil || len(ddlFiles) != 1 {
		t.Fatalf("schéma exporté : %v (err=%v)", ddlFiles, err)
	}
	truncated := ddlFiles[0]
	if err := os.WriteFile(truncated, []byte(`{"sequences": [], "tables": {"t": "CREATE TA`), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	res, err := ImportFromStaging(context.Background(), staging, func(k string) string {
		return filepath.Join(dir, k+".duckdb")
	}, nil, false)
	if err != nil {
		t.Fatalf("un schéma tronqué ne doit pas arrêter la restauration : %v", err)
	}
	if len(res.Restored) != 2 || len(res.FallbackTables) != 1 || res.FallbackTables[0] != "db1:t" {
		t.Fatalf("résultat : %+v", res)
	}
	db1 := openRestored(t, filepath.Join(dir, "db1.duckdb"))
	if n := scalar(t, db1, `SELECT count(*) FROM t`); n != 2 {
		t.Fatalf("lignes perdues : %d", n)
	}
	if countConstraints(t, openRestored(t, filepath.Join(dir, "db2.duckdb")), "u", "PRIMARY KEY") != 1 {
		t.Fatal("la clé voisine doit garder son schéma")
	}
}

// TestWriteSchemaDDL_EcritureAtomique : le schéma est écrit en entier ou pas du tout — aucun
// fichier partiel ni temporaire ne reste quand l'écriture échoue (répertoire absent).
func TestWriteSchemaDDL_EcritureAtomique(t *testing.T) {
	db := openRestored(t, filepath.Join(t.TempDir(), "s.duckdb"))
	if _, err := db.Exec(`CREATE TABLE t (v INTEGER PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	ok := filepath.Join(dir, ddlFilePrefix+"1"+ddlFileSuffix)
	writeSchemaDDL(context.Background(), db, "k", ok)
	if s, err := readSchemaFile(ok); err != nil || s == nil || s.Tables["t"] == "" {
		t.Fatalf("schéma écrit illisible : %+v (err=%v)", s, err)
	}
	missing := filepath.Join(dir, "absent")
	writeSchemaDDL(context.Background(), db, "k", filepath.Join(missing, ddlFilePrefix+"2"+ddlFileSuffix))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("fichiers restants après un échec d'écriture : %d, want 1", len(entries))
	}
}

// TestImport_SchemaDUnAutreExportIgnore : le staging garde deux cycles ; le dernier n'a pas
// écrit son schéma. Le schéma du cycle précédent ne décrit pas les lignes retenues : il est
// ignoré et la table passe par le repli (comptée), jamais rejouée avec un schéma d'une autre
// époque.
func TestImport_SchemaDUnAutreExportIgnore(t *testing.T) {
	staging := exportDB(t, "db1", `CREATE TABLE t (v INTEGER PRIMARY KEY); INSERT INTO t VALUES (1), (2);`)
	parquets, err := filepath.Glob(filepath.Join(staging, "db1", "t_*.parquet"))
	if err != nil || len(parquets) != 1 {
		t.Fatalf("export : %v (err=%v)", parquets, err)
	}
	// Dernier cycle : ses Parquet, sans le schéma qui aurait dû les accompagner.
	if err := os.Rename(parquets[0], filepath.Join(staging, "db1", "t_99991231_235959.parquet")); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "db1.duckdb")
	res := importTo(t, staging, target, nil)
	if len(res.FallbackTables) != 1 || res.FallbackTables[0] != "db1:t" {
		t.Fatalf("le schéma d'un autre export doit être ignoré : %+v", res)
	}
	if n := scalar(t, openRestored(t, target), `SELECT count(*) FROM t`); n != 2 {
		t.Fatalf("lignes perdues : %d", n)
	}
}

// TestImport_TableDUnCycleAnterieurRepli : une table dont le dernier export a échoué garde son
// Parquet d'un cycle antérieur ; le schéma du dernier cycle ne la décrit pas à cette époque :
// elle passe par le repli, les autres tables gardent le schéma.
func TestImport_TableDUnCycleAnterieurRepli(t *testing.T) {
	staging := exportDB(t, "db1", `CREATE TABLE t (v INTEGER PRIMARY KEY); INSERT INTO t VALUES (1);
		CREATE TABLE u (v INTEGER PRIMARY KEY); INSERT INTO u VALUES (1);`)
	us, err := filepath.Glob(filepath.Join(staging, "db1", "u_*.parquet"))
	if err != nil || len(us) != 1 {
		t.Fatalf("export : %v (err=%v)", us, err)
	}
	if err := os.Rename(us[0], filepath.Join(staging, "db1", "u_20000101_000000.parquet")); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "db1.duckdb")
	res := importTo(t, staging, target, nil)
	if len(res.FallbackTables) != 1 || res.FallbackTables[0] != "db1:u" {
		t.Fatalf("FallbackTables = %v, want [db1:u]", res.FallbackTables)
	}
	if countConstraints(t, openRestored(t, target), "t", "PRIMARY KEY") != 1 {
		t.Fatal("la table du dernier cycle doit garder son schéma")
	}
}

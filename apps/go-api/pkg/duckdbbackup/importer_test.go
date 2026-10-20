package duckdbbackup

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// ddlSchema : schéma canonique de test — un script DDL par clé, ErrNoCanonicalSchema sinon.
type ddlSchema struct {
	ddl       map[string]string
	finalized []string
}

func (s *ddlSchema) Provision(ctx context.Context, key string, db *sql.DB) error {
	ddl, ok := s.ddl[key]
	if !ok {
		return ErrNoCanonicalSchema
	}
	_, err := db.ExecContext(ctx, ddl)
	return err
}

func (s *ddlSchema) Finalize(_ context.Context, key string, _ *sql.DB) error {
	s.finalized = append(s.finalized, key)
	return nil
}

const testCanonicalDDL = `
	CREATE TABLE schema_migrations (name VARCHAR PRIMARY KEY, applied_at TIMESTAMP);
	INSERT INTO schema_migrations VALUES ('step_du_provisionnement', now());
	CREATE TABLE medals (match_id VARCHAR, xuid VARCHAR, n INTEGER DEFAULT 0, PRIMARY KEY (match_id, xuid));
	CREATE TABLE aliases (xuid VARCHAR, gamertag VARCHAR, UNIQUE (gamertag));
`

// exportDB crée une base source, l'exporte en Parquet sous staging/<key>/ et rend le dossier
// de staging.
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

// TestImport_RestaureLesClesDuSchemaCanonique : export puis import d'une base à clés → les
// PRIMARY KEY et UNIQUE sont présentes après restauration, les lignes de la sauvegarde
// remplacent celles du provisionnement, et un INSERT … ON CONFLICT de prod réussit.
func TestImport_RestaureLesClesDuSchemaCanonique(t *testing.T) {
	staging := exportDB(t, "db1", `
		CREATE TABLE schema_migrations (name VARCHAR PRIMARY KEY, applied_at TIMESTAMP);
		INSERT INTO schema_migrations VALUES ('step_a', '2026-01-01'), ('step_b', '2026-01-02');
		CREATE TABLE medals (match_id VARCHAR, xuid VARCHAR, n INTEGER DEFAULT 0, PRIMARY KEY (match_id, xuid));
		INSERT INTO medals VALUES ('m1', 'x1', 3), ('m1', 'x2', 1);
		CREATE TABLE aliases (xuid VARCHAR, gamertag VARCHAR, UNIQUE (gamertag));
		INSERT INTO aliases VALUES ('x1', 'A');
	`)
	target := filepath.Join(t.TempDir(), "out", "db1.duckdb")
	schema := &ddlSchema{ddl: map[string]string{"db1": testCanonicalDDL}}
	res, err := ImportFromStaging(context.Background(), staging,
		func(string) string { return target }, schema, false)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if len(res.Restored) != 1 || len(res.FallbackTables) != 0 || res.DroppedRows != 0 {
		t.Fatalf("résultat inattendu : %+v", res)
	}
	if len(schema.finalized) != 1 {
		t.Fatalf("Finalize non appelé : %v", schema.finalized)
	}

	db := openRestored(t, target)
	for _, c := range []struct{ table, kind string }{
		{"schema_migrations", "PRIMARY KEY"}, {"medals", "PRIMARY KEY"}, {"aliases", "UNIQUE"},
	} {
		if countConstraints(t, db, c.table, c.kind) != 1 {
			t.Errorf("%s : contrainte %s absente après restauration", c.table, c.kind)
		}
	}
	var names int
	if err := db.QueryRow(`SELECT count(*) FROM schema_migrations WHERE name IN ('step_a', 'step_b')`).Scan(&names); err != nil || names != 2 {
		t.Fatalf("registre de la sauvegarde non restauré : %d (err=%v)", names, err)
	}
	var prov int
	if err := db.QueryRow(`SELECT count(*) FROM schema_migrations WHERE name = 'step_du_provisionnement'`).Scan(&prov); err != nil || prov != 0 {
		t.Fatalf("les lignes du provisionnement doivent être remplacées : %d (err=%v)", prov, err)
	}
	// Écritures de prod sur la base restaurée : elles exigent la clé.
	if _, err := db.Exec(`INSERT INTO medals (match_id, xuid, n) VALUES ('m1', 'x1', 9)
		ON CONFLICT (match_id, xuid) DO NOTHING`); err != nil {
		t.Fatalf("ON CONFLICT sur la base restaurée : %v", err)
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO aliases VALUES ('x9', 'A')`); err != nil {
		t.Fatalf("INSERT OR IGNORE sur la base restaurée : %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT n FROM medals WHERE match_id = 'm1' AND xuid = 'x1'`).Scan(&n); err != nil || n != 3 {
		t.Fatalf("ligne restaurée altérée : n=%d err=%v", n, err)
	}
	// Le défaut du schéma canonique est présent (un CREATE TABLE AS l'aurait perdu).
	if _, err := db.Exec(`INSERT INTO medals (match_id, xuid) VALUES ('m2', 'x1')`); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT n FROM medals WHERE match_id = 'm2'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("défaut canonique absent : n=%d err=%v", n, err)
	}
}

// TestImport_DoublonsEcartesColonneConserveeEtRepli : une sauvegarde d'une base qui avait perdu
// ses clés (doublons), une colonne inconnue du schéma et une table hors schéma.
func TestImport_DoublonsEcartesColonneConserveeEtRepli(t *testing.T) {
	// Sauvegarde antérieure au DDL des tables : la table hors schéma passe par le repli.
	staging := exportDB(t, "db1", `
		CREATE TABLE schema_migrations (name VARCHAR, applied_at TIMESTAMP);
		INSERT INTO schema_migrations VALUES ('s', '2026-01-01'), ('s', '2026-03-01');
		CREATE TABLE medals (match_id VARCHAR, xuid VARCHAR, n INTEGER, legacy VARCHAR);
		INSERT INTO medals VALUES ('m1', 'x1', 3, 'old');
		CREATE TABLE aliases (xuid VARCHAR, gamertag VARCHAR);
		CREATE TABLE legacy_only (v INTEGER);
		INSERT INTO legacy_only VALUES (1);
	`)
	removeDDL(t, staging)
	target := filepath.Join(t.TempDir(), "db1.duckdb")
	res, err := ImportFromStaging(context.Background(), staging, func(string) string { return target },
		&ddlSchema{ddl: map[string]string{"db1": testCanonicalDDL}}, false)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if res.DroppedRows != 1 {
		t.Errorf("DroppedRows = %d, want 1", res.DroppedRows)
	}
	if len(res.FallbackTables) != 1 || res.FallbackTables[0] != "db1:legacy_only" {
		t.Errorf("FallbackTables = %v", res.FallbackTables)
	}
	db := openRestored(t, target)
	var applied string
	if err := db.QueryRow(`SELECT CAST(applied_at AS VARCHAR) FROM schema_migrations WHERE name = 's'`).Scan(&applied); err != nil {
		t.Fatal(err)
	}
	if applied != "2026-03-01 00:00:00" {
		t.Errorf("doublon : la ligne la plus récente doit être gardée, got %s", applied)
	}
	var legacy string
	if err := db.QueryRow(`SELECT legacy FROM medals`).Scan(&legacy); err != nil || legacy != "old" {
		t.Errorf("colonne hors schéma perdue : %q (err=%v)", legacy, err)
	}
}

// TestImport_SansSchemaCanoniqueRepliJournalise : clé sans schéma connu → toutes les tables
// passent par le repli, comptées.
func TestImport_SansSchemaCanoniqueRepliJournalise(t *testing.T) {
	staging := exportDB(t, "other", `CREATE TABLE t (v INTEGER PRIMARY KEY); INSERT INTO t VALUES (1);`)
	removeDDL(t, staging)
	target := filepath.Join(t.TempDir(), "o.duckdb")
	res, err := ImportFromStaging(context.Background(), staging, func(string) string { return target },
		&ddlSchema{ddl: map[string]string{}}, false)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if len(res.FallbackTables) != 1 {
		t.Fatalf("FallbackTables = %v", res.FallbackTables)
	}
}

// TestImport_EchecRemetLaBaseEnPlace : un schéma qui échoue laisse l'ancienne base intacte.
func TestImport_EchecRemetLaBaseEnPlace(t *testing.T) {
	staging := exportDB(t, "db1", `CREATE TABLE t (v INTEGER); INSERT INTO t VALUES (1);`)
	target := filepath.Join(t.TempDir(), "db1.duckdb")
	if err := os.WriteFile(target, []byte("ancienne"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ImportFromStaging(context.Background(), staging, func(string) string { return target },
		&ddlSchema{ddl: map[string]string{"db1": "CREATE TABLE ("}}, false)
	if err == nil {
		t.Fatal("un schéma en échec doit faire échouer l'import")
	}
	got, readErr := os.ReadFile(target)
	if readErr != nil || string(got) != "ancienne" {
		t.Fatalf("ancienne base non remise en place : %q (err=%v)", got, readErr)
	}
}

func TestImport_SchemaRequis(t *testing.T) {
	if _, err := ImportFromStaging(context.Background(), t.TempDir(), func(string) string { return "" }, nil, false); err == nil {
		t.Fatal("un import sans schéma canonique doit être refusé")
	}
}

// TestImport_NullConservesSurColonneNotNull : une sauvegarde porte des NULL dans une colonne que
// le schéma canonique déclare NOT NULL avec un défaut (cas des `written_at` des lignes anciennes
// d'une table append-only) : les NULL sont restaurés tels quels, jamais remplacés par le défaut.
func TestImport_NullConservesSurColonneNotNull(t *testing.T) {
	staging := exportDB(t, "db1", `
		CREATE TABLE snaps (id BIGINT PRIMARY KEY, v INTEGER, written_at TIMESTAMP);
		INSERT INTO snaps VALUES (1, 10, NULL), (2, 20, '2026-05-01');
	`)
	target := filepath.Join(t.TempDir(), "db1.duckdb")
	ddl := `CREATE TABLE snaps (id BIGINT PRIMARY KEY, v INTEGER,
		written_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP));`
	if _, err := ImportFromStaging(context.Background(), staging, func(string) string { return target },
		&ddlSchema{ddl: map[string]string{"db1": ddl}}, false); err != nil {
		t.Fatalf("import: %v", err)
	}
	db := openRestored(t, target)
	var nulls int
	if err := db.QueryRow(`SELECT count(*) FROM snaps WHERE written_at IS NULL`).Scan(&nulls); err != nil || nulls != 1 {
		t.Fatalf("NULL non conservé : %d (err=%v)", nulls, err)
	}
	if countConstraints(t, db, "snaps", "PRIMARY KEY") != 1 {
		t.Fatal("clé primaire absente")
	}
}

// TestImport_TableHorsSchemaRecreeDepuisLeDDL : une table que le schéma canonique ne connaît pas
// (créée hors migrations par un module) retrouve ses clés grâce au DDL que porte la sauvegarde ;
// seul le dernier export d'une table fait foi.
func TestImport_TableHorsSchemaRecreeDepuisLeDDL(t *testing.T) {
	staging := exportDB(t, "db1", `
		CREATE TABLE runtime_only (k VARCHAR PRIMARY KEY, v INTEGER NOT NULL DEFAULT 7);
		INSERT INTO runtime_only VALUES ('a', 1);
	`)
	// Un export plus ancien de la même table, qui ne doit pas être importé.
	old := filepath.Join(staging, "db1", "runtime_only_20000101_000000.parquet")
	db0, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db0.Exec(`COPY (SELECT 'zz' AS k, 0 AS v) TO '` + filepath.ToSlash(old) + `' (FORMAT PARQUET)`); err != nil {
		t.Fatal(err)
	}
	_ = db0.Close()

	target := filepath.Join(t.TempDir(), "db1.duckdb")
	res, err := ImportFromStaging(context.Background(), staging, func(string) string { return target },
		&ddlSchema{ddl: map[string]string{"db1": testCanonicalDDL}}, false)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if len(res.FallbackTables) != 0 || len(res.SourceDDLTables) != 1 {
		t.Fatalf("résultat : %+v", res)
	}
	db := openRestored(t, target)
	if countConstraints(t, db, "runtime_only", "PRIMARY KEY") != 1 {
		t.Fatal("clé de la table hors schéma non restaurée")
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM runtime_only WHERE k = 'a'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("dernier export non importé : %d (err=%v)", n, err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM runtime_only`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("un export ancien a été importé : %d lignes (err=%v)", n, err)
	}
	if _, err := db.Exec(`INSERT INTO runtime_only (k) VALUES ('a') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("ON CONFLICT : %v", err)
	}
}

// removeDDL retire le DDL des tables du staging (sauvegarde antérieure à son introduction).
func removeDDL(t *testing.T, staging string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(staging, "*", ddlFilePrefix+"*"))
	if err != nil || len(matches) == 0 {
		t.Fatalf("DDL introuvable dans le staging (err=%v)", err)
	}
	for _, m := range matches {
		if err := os.Remove(m); err != nil {
			t.Fatal(err)
		}
	}
}

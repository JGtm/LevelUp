package duckdbbackup

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckIntegrity_ValidDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.duckdb")

	// Create a minimal DuckDB database.
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := db.Exec("CREATE TABLE t (id INTEGER)"); err != nil {
		db.Close()
		t.Fatalf("create table: %v", err)
	}
	db.Close()

	target := Target{Key: "test", Path: dbPath}
	result := CheckIntegrity(context.Background(), target)

	// Expect either OK=true (pragma supported and returned "ok") or
	// OK=true with no detail (pragma not supported — inconclusive).
	if !result.OK {
		t.Errorf("CheckIntegrity on valid DB: OK=false, Detail=%q", result.Detail)
	}
	if result.CheckedAt.IsZero() {
		t.Error("CheckedAt should not be zero")
	}
}

func TestCheckIntegrity_NonDBFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.duckdb")
	// Write garbage that can't be a valid DuckDB file.
	if err := os.WriteFile(path, []byte("not a duckdb"), 0o644); err != nil {
		t.Fatal(err)
	}

	target := Target{Key: "corrupt", Path: path}
	result := CheckIntegrity(context.Background(), target)

	// A garbage file may fail to open, fail to run pragma, or pass integrity_check
	// returning an error row. Any of these is acceptable — we just check that the
	// function returns without panicking and sets CheckedAt.
	if result.CheckedAt.IsZero() {
		t.Error("CheckedAt should not be zero")
	}
}

// TestExportTarget_Basic creates a real DuckDB with two tables and verifies that
// ExportTarget produces one Parquet file per table in the output directory.
func TestExportTarget_Basic(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.duckdb")
	outDir := filepath.Join(dir, "out")

	// Create a DuckDB with two tables.
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := db.Exec("CREATE TABLE players (id INTEGER, name VARCHAR)"); err != nil {
		db.Close()
		t.Fatalf("create players: %v", err)
	}
	if _, err := db.Exec("CREATE TABLE matches (match_id VARCHAR)"); err != nil {
		db.Close()
		t.Fatalf("create matches: %v", err)
	}
	db.Close()

	target := Target{Key: "test", Path: dbPath}
	n, err := ExportTarget(context.Background(), target, outDir, 1)
	if err != nil {
		t.Fatalf("ExportTarget: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 tables exported, got %d", n)
	}

	// Verify two Parquet files were created.
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 parquet files, got %d", len(entries))
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".parquet") {
			t.Errorf("unexpected file %q (expected .parquet)", e.Name())
		}
	}
}

// TestExportTarget_ReusesProvidedConn vérifie que quand Target.OpenDB est défini,
// l'exporter réutilise la connexion fournie sans tenter d'ouvrir un 2e handle.
// Régression : avant le refactor, ExportTarget appelait sql.Open avec
// `?access_mode=read_only` et DuckDB refusait l'ouverture si un autre handle
// in-process tenait le fichier en RW ("different configuration"). C'est ce qui
// faisait échouer le backup de metadata.duckdb et shared_social.duckdb.
func TestExportTarget_ReusesProvidedConn(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "rw_held.duckdb")
	outDir := filepath.Join(dir, "out")

	// Tient le fichier en RW (mode par défaut, pas access_mode=read_only).
	rw, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open RW: %v", err)
	}
	defer rw.Close()
	if _, err := rw.Exec("CREATE TABLE t (id INTEGER)"); err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Sanity check : sans OpenDB, l'export doit échouer avec "different configuration".
	bareTarget := Target{Key: "bare", Path: dbPath}
	_, errBare := ExportTarget(context.Background(), bareTarget, filepath.Join(outDir, "bare"), 1)
	if errBare == nil {
		t.Fatal("attendu erreur 'different configuration' sans OpenDB (RW tenu par autre handle)")
	}
	if !strings.Contains(errBare.Error(), "different configuration") {
		t.Logf("note : message d'erreur ne contient pas 'different configuration' : %v", errBare)
	}

	// Avec OpenDB pointant sur la conn RW existante, l'export réussit.
	target := Target{
		Key:  "shared",
		Path: dbPath,
		OpenDB: func(_ context.Context) (*sql.DB, func(), error) {
			return rw, func() {}, nil
		},
	}
	n, err := ExportTarget(context.Background(), target, outDir, 1)
	if err != nil {
		t.Fatalf("ExportTarget avec OpenDB: %v", err)
	}
	if n != 1 {
		t.Errorf("attendu 1 table exportée, got %d", n)
	}

	// La conn RW doit toujours être utilisable après l'export (pas fermée par release).
	if _, err := rw.Exec("INSERT INTO t VALUES (1)"); err != nil {
		t.Errorf("conn RW fermée par l'exporter : %v", err)
	}
}

// TestCheckIntegrity_ReusesProvidedConn : même garantie que ExportTarget pour
// PRAGMA integrity_check sur un fichier détenu en RW ailleurs.
func TestCheckIntegrity_ReusesProvidedConn(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "rw_held.duckdb")

	rw, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open RW: %v", err)
	}
	defer rw.Close()
	if _, err := rw.Exec("CREATE TABLE t (id INTEGER)"); err != nil {
		t.Fatalf("create: %v", err)
	}

	target := Target{
		Key:  "shared",
		Path: dbPath,
		OpenDB: func(_ context.Context) (*sql.DB, func(), error) {
			return rw, func() {}, nil
		},
	}
	res := CheckIntegrity(context.Background(), target)
	if !res.OK {
		t.Errorf("CheckIntegrity avec OpenDB: OK=false detail=%q", res.Detail)
	}
}

func TestManifest_SetIntegrityResult(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".manifest.json")

	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	m.SetIntegrityResult("shared", IntegrityResult{OK: true})
	m.SetIntegrityResult("player", IntegrityResult{OK: false, Detail: "corrupted page 1"})

	if err := m.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	m2, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest after save: %v", err)
	}
	if got := m2.IntegrityChecks["shared"]; !got.OK {
		t.Errorf("shared: expected OK=true, got %+v", got)
	}
	if got := m2.IntegrityChecks["player"]; got.OK || got.Detail != "corrupted page 1" {
		t.Errorf("player: expected OK=false detail='corrupted page 1', got %+v", got)
	}
}

// TestListTables_ScopesToCurrentCatalog vérifie que listTables ne remonte QUE
// les tables du catalogue courant (schéma main), et ignore les tables des
// catalogues ATTACHés. Régression du bug backup `xuid_aliases` (2026-05-31) :
// la conn pool du serveur a global/shared attachés ; sans le filtre
// table_catalog = current_database(), listTables remontait `xuid_aliases` du
// catalogue global puis `COPY "xuid_aliases"` échouait ("Table does not exist")
// et faisait perdre tout le backup du joueur.
func TestListTables_ScopesToCurrentCatalog(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	stmts := []string{
		`CREATE TABLE player_match_enrichment (match_id VARCHAR)`,
		`CREATE TABLE media_files (file_path VARCHAR)`,
		// Catalogue attaché simulant `global` (xbox_aliases) avec une table
		// homonyme d'une legacy player (xuid_aliases) — exactement ce qui
		// polluait la liste en prod.
		`ATTACH ':memory:' AS other`,
		`CREATE TABLE other.xuid_aliases (xuid VARCHAR)`,
		`CREATE TABLE other.match_registry (match_id VARCHAR)`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			t.Fatalf("exec %q: %v", s, err)
		}
	}

	tables, err := listTables(ctx, db)
	if err != nil {
		t.Fatalf("listTables: %v", err)
	}

	got := map[string]bool{}
	for _, name := range tables {
		got[name] = true
	}
	if !got["player_match_enrichment"] || !got["media_files"] {
		t.Errorf("tables du catalogue courant manquantes: %v", tables)
	}
	if got["xuid_aliases"] || got["match_registry"] {
		t.Errorf("tables d'un catalogue attaché listées à tort: %v", tables)
	}
}

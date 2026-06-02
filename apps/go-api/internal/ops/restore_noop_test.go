//go:build integration

// Package ops — restore_noop_test.go : non-régression du faux "Success" silencieux
// de RestorePlayer (revue P0 2026-06-02). Réutilise les helpers de restore_test.go
// (createTestPlayerDB, exportTestParquet). CGO requis : go test -tags=integration.
package ops

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestRestorePlayer_NonEmptyTableNoReplace_Refuses : restaurer par-dessus une table
// déjà peuplée sans --replace doit échouer explicitement — avant le fix, un
// CREATE TABLE IF NOT EXISTS AS SELECT était un no-op silencieux retournant
// Success=true sans rien restaurer.
func TestRestorePlayer_NonEmptyTableNoReplace_Refuses(t *testing.T) {
	dir := t.TempDir()
	srcDBPath := filepath.Join(dir, "src.duckdb")
	dstDBPath := filepath.Join(dir, "dst.duckdb")
	backupDir := filepath.Join(dir, "backup")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatal(err)
	}

	createTestPlayerDB(t, srcDBPath)
	exportTestParquet(t, srcDBPath, backupDir)

	// DB cible déjà peuplée (3 rows, même schéma).
	createTestPlayerDB(t, dstDBPath)

	res, err := RestorePlayer(context.Background(), RestoreOptions{
		PlayerDBPath: dstDBPath,
		BackupDir:    backupDir,
		Replace:      false,
	})
	if err == nil {
		t.Fatalf("restore sans --replace sur table non vide doit échouer (pas de faux succès) ; result=%+v", res)
	}
	if !strings.Contains(err.Error(), "non vide") {
		t.Errorf("message d'erreur inattendu: %v", err)
	}
}

// TestRestorePlayer_EmptyMigratedTableNoReplace_Loads : restaurer dans une table
// migrée vide (avec PK), sans --replace, charge bien les rows en préservant la PK.
func TestRestorePlayer_EmptyMigratedTableNoReplace_Loads(t *testing.T) {
	dir := t.TempDir()
	srcDBPath := filepath.Join(dir, "src.duckdb")
	dstDBPath := filepath.Join(dir, "dst.duckdb")
	backupDir := filepath.Join(dir, "backup")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatal(err)
	}

	createTestPlayerDB(t, srcDBPath)
	exportTestParquet(t, srcDBPath, backupDir)

	// DB cible : table vide AVEC PK (état après migrations).
	dst, err := sql.Open("duckdb", dstDBPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dst.Exec(`CREATE TABLE player_match_enrichment (
		match_id VARCHAR PRIMARY KEY,
		performance_score FLOAT,
		session_id VARCHAR
	)`); err != nil {
		t.Fatal(err)
	}
	dst.Close()

	res, err := RestorePlayer(context.Background(), RestoreOptions{
		PlayerDBPath: dstDBPath,
		BackupDir:    backupDir,
		Replace:      false,
	})
	if err != nil {
		t.Fatalf("restore dans table migrée vide: %v", err)
	}
	if !res.Success {
		t.Fatalf("attendu Success: %s", res.Message)
	}

	db, err := sql.Open("duckdb", dstDBPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM player_match_enrichment").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("attendu 3 rows chargées, obtenu %d", n)
	}
	// La PK doit avoir été préservée (INSERT dans la table migrée, pas CTAS).
	var pkCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM duckdb_constraints()
		WHERE table_name = 'player_match_enrichment' AND constraint_type = 'PRIMARY KEY'`).Scan(&pkCount); err != nil {
		t.Fatal(err)
	}
	if pkCount == 0 {
		t.Error("la PRIMARY KEY devrait être préservée après restauration par INSERT")
	}
}

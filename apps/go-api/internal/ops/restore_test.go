//go:build integration

// Package ops — restore_test.go : tests RestorePlayer depuis Parquet.
//
// Sprint 47 T23 — backup → restore → vérification intégrité.
// CGO requis : DuckDB pour lire et écrire les fichiers .duckdb.
// Lancer avec : go test -tags=integration ./internal/ops/ -v

package ops

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// createTestPlayerDB crée une DB player minimaliste avec quelques lignes.
func createTestPlayerDB(t *testing.T, dbPath string) {
	t.Helper()
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("créer DB test: %v", err)
	}
	defer db.Close()

	// Table minimale player_match_enrichment (sous-ensemble des colonnes)
	if _, err := db.Exec(`CREATE TABLE player_match_enrichment (
		match_id VARCHAR PRIMARY KEY,
		performance_score FLOAT,
		session_id VARCHAR
	)`); err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}
	for i, mid := range []string{"match-001", "match-002", "match-003"} {
		if _, err := db.Exec(
			"INSERT INTO player_match_enrichment VALUES (?, ?, ?)",
			mid, float64(i+1)*10.0, "session-A",
		); err != nil {
			t.Fatalf("INSERT %s: %v", mid, err)
		}
	}
}

// exportTestParquet exporte la table en Parquet via DuckDB COPY avec timestamp fixe.
func exportTestParquet(t *testing.T, dbPath, backupDir string) {
	t.Helper()
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("ouvrir DB pour export: %v", err)
	}
	defer db.Close()

	const ts = "20250101_120000"
	parqPath := filepath.Join(backupDir, "player_match_enrichment_"+ts+".parquet")
	if _, err := db.Exec("COPY player_match_enrichment TO ? (FORMAT PARQUET)", parqPath); err != nil {
		t.Fatalf("COPY TO PARQUET: %v", err)
	}
	// Créer un backup_metadata_*.json pour que findLatestParquetFiles trouve le timestamp.
	metaPath := filepath.Join(backupDir, "backup_metadata_"+ts+".json")
	if err := os.WriteFile(metaPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("écrire backup_metadata: %v", err)
	}
}

// TestRestorePlayer_DryRun vérifie qu'un dry run liste les fichiers sans modifier la DB.
func TestRestorePlayer_DryRun(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "stats.duckdb")
	backupDir := filepath.Join(dir, "backup")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatal(err)
	}

	createTestPlayerDB(t, dbPath)
	exportTestParquet(t, dbPath, backupDir)

	result, err := RestorePlayer(RestoreOptions{
		Gamertag:     "TestSpartan",
		PlayerDBPath: dbPath,
		BackupDir:    backupDir,
		DryRun:       true,
	})
	if err != nil {
		t.Fatalf("RestorePlayer DryRun: %v", err)
	}
	if !result.DryRun {
		t.Error("result.DryRun devrait être true")
	}
	if len(result.TablesLoaded) == 0 {
		t.Error("TablesLoaded vide en dry run — attendu au moins player_match_enrichment")
	}
	t.Logf("dry run OK : %v", result.TablesLoaded)
}

// TestRestorePlayer_RestoreAndVerify effectue une restauration réelle et vérifie les données.
func TestRestorePlayer_RestoreAndVerify(t *testing.T) {
	dir := t.TempDir()
	srcDBPath := filepath.Join(dir, "stats_src.duckdb")
	restoreDBPath := filepath.Join(dir, "stats_restored.duckdb")
	backupDir := filepath.Join(dir, "backup")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Créer et exporter la DB source
	createTestPlayerDB(t, srcDBPath)
	exportTestParquet(t, srcDBPath, backupDir)

	// Créer la DB cible vide avec le même schéma (RestorePlayer a besoin de la table)
	dstDB, err := sql.Open("duckdb", restoreDBPath)
	if err != nil {
		t.Fatalf("créer DB cible: %v", err)
	}
	if _, err := dstDB.Exec(`CREATE TABLE player_match_enrichment (
		match_id VARCHAR PRIMARY KEY,
		performance_score FLOAT,
		session_id VARCHAR
	)`); err != nil {
		t.Fatalf("CREATE TABLE cible: %v", err)
	}
	dstDB.Close()

	// Restaurer
	result, err := RestorePlayer(RestoreOptions{
		Gamertag:     "TestSpartan",
		PlayerDBPath: restoreDBPath,
		BackupDir:    backupDir,
		Replace:      true,
		DryRun:       false,
	})
	if err != nil {
		t.Fatalf("RestorePlayer: %v", err)
	}
	if !result.Success {
		t.Errorf("restauration échouée : %s", result.Message)
	}

	// Vérifier les 3 lignes sont présentes dans la DB restaurée
	db, err := sql.Open("duckdb", restoreDBPath)
	if err != nil {
		t.Fatalf("ouvrir DB restaurée: %v", err)
	}
	defer db.Close()

	var cnt int
	if err := db.QueryRow("SELECT COUNT(*) FROM player_match_enrichment").Scan(&cnt); err != nil {
		t.Fatalf("COUNT: %v", err)
	}
	if cnt != 3 {
		t.Errorf("attendu 3 lignes après restauration, obtenu %d", cnt)
	}
	t.Logf("✅ restauration vérifiée : %d lignes dans player_match_enrichment", cnt)
}

// TestFindAvailableBackups_EmptyDir vérifie qu'un répertoire vide retourne une liste vide.
func TestFindAvailableBackups_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	backups, err := FindAvailableBackups(dir)
	if err != nil {
		t.Fatalf("FindAvailableBackups: %v", err)
	}
	if len(backups) != 0 {
		t.Errorf("attendu 0 backups, obtenu %d", len(backups))
	}
}

// TestRestorePlayer_NoBackupFiles vérifie l'erreur quand aucun Parquet n'est présent.
func TestRestorePlayer_NoBackupFiles(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "stats.duckdb")
	emptyBackupDir := filepath.Join(dir, "empty")
	if err := os.MkdirAll(emptyBackupDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Créer une DB vide (RestorePlayer en a besoin)
	db, _ := sql.Open("duckdb", dbPath)
	db.Close()

	_, err := RestorePlayer(RestoreOptions{
		PlayerDBPath: dbPath,
		BackupDir:    emptyBackupDir,
	})
	if err == nil {
		t.Error("attendu une erreur quand aucun fichier Parquet n'est présent")
	}
}

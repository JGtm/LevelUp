package duckdb_test

import (
	"path/filepath"
	"testing"

	ddb "levelup/go-api/internal/platform/duckdb"
)

// TestPoolStatsSnapshot vérifie que le handle ouvert apparaît dans le snapshot
// d'observabilité (J1) avec la limite de pool attendue. Chemin de fichier unique
// (t.TempDir) → clé de cache unique, robuste face aux DB ouvertes par d'autres tests.
func TestPoolStatsSnapshot(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "pool_stats.duckdb")
	db, err := ddb.OpenReadWriteShared(dbPath) // pool partagé : MaxOpenConns = 4
	if err != nil {
		t.Fatalf("OpenReadWriteShared: %v", err)
	}
	defer db.Close()

	snap := ddb.PoolStatsSnapshot()
	key := "rw:" + dbPath
	stats, ok := snap[key]
	if !ok {
		t.Fatalf("handle %q absent du snapshot (%d entrées)", key, len(snap))
	}
	if stats.MaxOpenConnections != 4 {
		t.Errorf("MaxOpenConnections = %d, want 4 (poolMaxOpenShared)", stats.MaxOpenConnections)
	}
}

// TestPoolStatsSnapshot_ExcludesClosed : un handle fermé ne figure plus au snapshot.
func TestPoolStatsSnapshot_ExcludesClosed(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "pool_closed.duckdb")
	db, err := ddb.OpenReadWriteShared(dbPath)
	if err != nil {
		t.Fatalf("OpenReadWriteShared: %v", err)
	}
	key := "rw:" + dbPath
	if _, ok := ddb.PoolStatsSnapshot()[key]; !ok {
		t.Fatalf("handle %q attendu avant fermeture", key)
	}
	_ = db.Close()
	if _, ok := ddb.PoolStatsSnapshot()[key]; ok {
		t.Errorf("handle %q toujours présent après Close", key)
	}
}

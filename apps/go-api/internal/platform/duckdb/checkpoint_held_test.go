package duckdb

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/platform/dblease"
)

// openWithWAL ouvre une base fichier en RW, y écrit une ligne et rend la base : tant qu'aucun
// CHECKPOINT n'est passé, la ligne vit dans le fichier .wal voisin.
func openWithWAL(t *testing.T, name string) (*DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	db, err := OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.loadSQL().Exec(`CREATE TABLE t (id INTEGER)`); err != nil {
		t.Fatalf("CREATE: %v", err)
	}
	if _, err := db.loadSQL().Exec(`INSERT INTO t VALUES (1), (2), (3)`); err != nil {
		t.Fatalf("INSERT: %v", err)
	}
	return db, path
}

func walSize(path string) int64 {
	info, err := os.Stat(path + ".wal")
	if err != nil {
		return 0
	}
	return info.Size()
}

func metricCount(key string) int64 {
	v := metricPeriodicCheckpoint.Get(key)
	if v == nil {
		return 0
	}
	n, _ := v.(interface{ Value() int64 })
	if n == nil {
		return 0
	}
	return n.Value()
}

// TestCheckpointHeld_EmptiesTheWAL : après une écriture puis le CHECKPOINT périodique, aucun
// fichier .wal ne reste.
func TestCheckpointHeld_EmptiesTheWAL(t *testing.T) {
	db, path := openWithWAL(t, "a.duckdb")
	if walSize(path) == 0 {
		t.Skip("DuckDB a déjà vidé le WAL de lui-même : rien à démontrer")
	}
	okBefore := metricCount("ok")
	report := CheckpointHeld(context.Background(), nil)
	if report.Done < 1 || report.Failed != 0 {
		t.Fatalf("rapport = %+v ; attendu au moins une base checkpointée, aucun échec", report)
	}
	if got := walSize(path); got != 0 {
		t.Errorf("WAL de %d octets après CHECKPOINT ; attendu aucun", got)
	}
	if metricCount("ok") <= okBefore {
		t.Error("la métrique ok n'a pas augmenté")
	}
	var n int
	if err := db.loadSQL().QueryRow(`SELECT count(*) FROM t`).Scan(&n); err != nil || n != 3 {
		t.Errorf("les données doivent survivre : %d, %v", n, err)
	}
}

// TestCheckpointHeld_SkipsABusyDatabase : un lot d'écriture qui tient le bail empêche le
// CHECKPOINT, la base est sautée et comptée, le WAL est intact.
func TestCheckpointHeld_SkipsABusyDatabase(t *testing.T) {
	_, path := openWithWAL(t, "b.duckdb")
	release, err := dblease.AcquireLease(path, time.Second)
	if err != nil {
		t.Fatalf("AcquireLease: %v", err)
	}
	defer release()
	before := walSize(path)
	busyBefore := metricCount("skipped_busy")

	report := CheckpointHeld(context.Background(), nil)
	if report.Busy < 1 {
		t.Fatalf("rapport = %+v ; attendu la base sautée", report)
	}
	if walSize(path) != before {
		t.Error("le WAL a changé alors qu'un lot d'écriture tenait le bail")
	}
	if metricCount("skipped_busy") <= busyBefore {
		t.Error("la métrique skipped_busy n'a pas augmenté")
	}
}

// TestCheckpointHeld_SkipListLeavesAPathAlone : un chemin de skip n'est pas touché.
func TestCheckpointHeld_SkipListLeavesAPathAlone(t *testing.T) {
	_, path := openWithWAL(t, "c.duckdb")
	before := walSize(path)
	CheckpointHeld(context.Background(), map[string]bool{path: true})
	if walSize(path) != before {
		t.Error("le chemin de skip a été checkpointé")
	}
}

// TestCheckpointHeld_AFailureIsCountedAndNeverSwallowed : un CHECKPOINT en erreur est compté
// (failed) et rapporté, sans panique ni arrêt du passage.
func TestCheckpointHeld_AFailureIsCountedAndNeverSwallowed(t *testing.T) {
	db, _ := openWithWAL(t, "d.duckdb")
	if err := db.loadSQL().Close(); err != nil { // handle mort, entrée de cache encore là
		t.Fatalf("Close: %v", err)
	}
	failedBefore := metricCount("failed")
	report := CheckpointHeld(context.Background(), nil)
	if report.Failed < 1 {
		t.Fatalf("rapport = %+v ; attendu un échec rapporté", report)
	}
	if metricCount("failed") <= failedBefore {
		t.Error("la métrique failed n'a pas augmenté")
	}
}

// TestHeldRWHandles_ListsTheOpenRWHandle : le handle RW ouvert figure dans la liste des bases à checkpointer.
func TestHeldRWHandles_ListsTheOpenRWHandle(t *testing.T) {
	_, path := openWithWAL(t, "e.duckdb")
	for _, db := range heldRWHandles(nil) {
		if db.path == path {
			return
		}
	}
	t.Fatal("le handle RW ouvert n'est pas listé")
}

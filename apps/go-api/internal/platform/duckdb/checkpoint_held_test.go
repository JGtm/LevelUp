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

// markClosed pose l'état « fermé » du handle comme le fait releaseWriter (le B-swap), sans fermer
// le handle SQL ; le rend à la fin du test pour que le nettoyage ferme vraiment la base.
func markClosed(t *testing.T, db *DB) {
	t.Helper()
	db.closed.Store(true)
	t.Cleanup(func() { db.closed.Store(false) })
}

// TestCheckpointHeld_ClosedAfterTheListIsSkippedNotFailed : la liste est figée avant le bail ;
// releaseWriter ferme le handle AVANT de rendre le bail. Un handle fermé pendant l'attente du bail
// est sauté et compté (skipped_closed), jamais compté en échec.
func TestCheckpointHeld_ClosedAfterTheListIsSkippedNotFailed(t *testing.T) {
	db, path := openWithWAL(t, "f.duckdb")
	release, err := dblease.AcquireLease(path, time.Second)
	if err != nil {
		t.Fatalf("AcquireLease: %v", err)
	}
	closedBefore, failedBefore := metricCount("skipped_closed"), metricCount("failed")

	reports := make(chan CheckpointReport, 1)
	go func() { reports <- CheckpointHeld(context.Background(), nil) }()
	// La base est déjà listée et attend le bail : le B-swap ferme le handle, puis rend le bail.
	time.Sleep(300 * time.Millisecond)
	markClosed(t, db)
	release()

	report := <-reports
	if report.Closed != 1 || report.Done != 0 || report.Failed != 0 {
		t.Fatalf("rapport = %+v ; attendu une base sautée comme fermée, ni faite ni en échec", report)
	}
	if metricCount("skipped_closed") != closedBefore+1 {
		t.Error("la métrique skipped_closed n'a pas augmenté")
	}
	if metricCount("failed") != failedBefore {
		t.Error("un handle fermé ne doit pas être compté en échec")
	}
}

// TestHeldRWHandles_DropsAClosedHandle : un handle déjà fermé n'est pas listé.
func TestHeldRWHandles_DropsAClosedHandle(t *testing.T) {
	db, path := openWithWAL(t, "g.duckdb")
	markClosed(t, db)
	for _, h := range heldRWHandles(nil) {
		if h.path == path {
			t.Fatal("un handle fermé est listé")
		}
	}
}

// TestCheckpointHeld_ACancelledContextStopsThePass : le contexte annulé (arrêt du serveur) arrête le
// passage avant toute base : rien n'est checkpointé, rien n'est compté en échec.
func TestCheckpointHeld_ACancelledContextStopsThePass(t *testing.T) {
	_, path := openWithWAL(t, "h.duckdb")
	before := walSize(path)
	failedBefore := metricCount("failed")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report := CheckpointHeld(ctx, nil)
	if report.Done+report.Failed+report.Busy+report.Closed+report.Interrupted != 0 {
		t.Errorf("rapport = %+v ; attendu aucun traitement", report)
	}
	if walSize(path) != before || metricCount("failed") != failedBefore {
		t.Error("le passage annulé a touché la base ou compté un échec")
	}
}

// TestCheckpointOne_InterruptedByShutdownIsCountedApart : un CHECKPOINT coupé par l'annulation du
// contexte (arrêt) est compté `interrupted`, pas `failed` : ce n'est pas une panne de la base.
func TestCheckpointOne_InterruptedByShutdownIsCountedApart(t *testing.T) {
	db, _ := openWithWAL(t, "i.duckdb")
	interruptedBefore, failedBefore := metricCount("interrupted"), metricCount("failed")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := checkpointOne(ctx, db); got != checkpointInterrupted {
		t.Fatalf("checkpointOne = %v ; attendu checkpointInterrupted", got)
	}
	if metricCount("interrupted") != interruptedBefore+1 {
		t.Error("la métrique interrupted n'a pas augmenté")
	}
	if metricCount("failed") != failedBefore {
		t.Error("un CHECKPOINT interrompu par l'arrêt ne doit pas être compté en échec")
	}
}

// TestCheckpointHeld_NeverTouchesAReadOnlyHandle : un handle ouvert en lecture seule n'est pas
// listé et n'est jamais checkpointé (DuckDB le refuserait, et un second process peut l'écrire).
func TestCheckpointHeld_NeverTouchesAReadOnlyHandle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ro.duckdb")
	rw, err := OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	if _, err := rw.loadSQL().Exec(`CREATE TABLE t (id INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if err := rw.Close(); err != nil { // le fichier existe, plus aucun handle RW
		t.Fatalf("Close: %v", err)
	}
	ro, err := OpenReadOnly(path)
	if err != nil {
		t.Fatalf("OpenReadOnly: %v", err)
	}
	t.Cleanup(func() { _ = ro.Close() })

	for _, h := range heldRWHandles(nil) {
		if h.path == path {
			t.Fatal("un handle en lecture seule est listé parmi les handles RW")
		}
	}
	failedBefore, okBefore := metricCount("failed"), metricCount("ok")
	report := CheckpointHeld(context.Background(), nil)
	if report.Failed != 0 || metricCount("failed") != failedBefore || metricCount("ok") != okBefore {
		t.Errorf("rapport = %+v ; le handle en lecture seule ne doit donner ni échec ni CHECKPOINT", report)
	}
}

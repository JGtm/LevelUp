package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/duckdb"
)

// TestPeriodicCheckpointTask_EmptiesTheWALOfAHeldDatabase : la tâche vide le WAL d'une base
// tenue en RW, laisse celle de skip, et s'arrête à l'annulation du contexte.
func TestPeriodicCheckpointTask_EmptiesTheWALOfAHeldDatabase(t *testing.T) {
	dir := t.TempDir()
	held := filepath.Join(dir, "held.duckdb")
	skipped := filepath.Join(dir, "skipped.duckdb")
	for _, p := range []string{held, skipped} {
		db, err := duckdb.OpenReadWrite(p)
		if err != nil {
			t.Fatalf("OpenReadWrite: %v", err)
		}
		t.Cleanup(func() { _ = db.Close() })
		if _, err := db.SQLDb().Exec(`CREATE TABLE t (id INTEGER)`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.SQLDb().Exec(`INSERT INTO t VALUES (1)`); err != nil {
			t.Fatal(err)
		}
	}
	wal := func(p string) int64 {
		info, err := os.Stat(p + ".wal")
		if err != nil {
			return 0
		}
		return info.Size()
	}
	skippedBefore := wal(skipped)

	ctx, cancel := context.WithCancel(context.Background())
	task := periodicCheckpointTask(ctx, 20*time.Millisecond, map[string]bool{skipped: true})
	if !task.tracked || task.name != taskPeriodicCkpt {
		t.Fatalf("tâche = %+v ; attendue suivie et nommée %q", task, taskPeriodicCkpt)
	}
	done := make(chan struct{})
	go func() { task.run(); close(done) }()

	deadline := time.Now().Add(10 * time.Second)
	for wal(held) != 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("la tâche ne s'arrête pas à l'annulation du contexte")
	}
	if got := wal(held); got != 0 {
		t.Errorf("WAL de la base tenue = %d octets ; attendu vidé", got)
	}
	if got := wal(skipped); got != skippedBefore {
		t.Errorf("WAL de la base à sauter changé (%d -> %d)", skippedBefore, got)
	}
}

// TestPeriodicCheckpoint_DeclaredLikeEveryTask : la tâche est déclarée par profil (tournée sauf en
// démo, où elle écrirait hors de la racine démo).
func TestPeriodicCheckpoint_DeclaredLikeEveryTask(t *testing.T) {
	if policyOf(taskPeriodicCkpt, config.DeploymentDesktop) != taskKeep ||
		policyOf(taskPeriodicCkpt, config.DeploymentDemo) != taskCut {
		t.Error("taskPeriodicCkpt : desktop doit la lancer, la démo la couper")
	}
}

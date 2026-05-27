//go:build cgo

// Package dblease — writer_commit_checkpoint_test.go (ADR 0021 Phase 3.2 bis).
//
// Tests de CommitWithCheckpoint : commit puis CHECKPOINT immédiat sur la DB.

package dblease

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openTestDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE t (id INTEGER, val VARCHAR);`); err != nil {
		t.Fatalf("create: %v", err)
	}
	return db, path
}

// TestCommitWithCheckpoint_FlushesWAL : insère dans une TX, commit avec
// checkpoint, ferme la DB sans CHECKPOINT supplémentaire, rouvre RO →
// la donnée doit être présente (preuve que CHECKPOINT a vidé le WAL).
func TestCommitWithCheckpoint_FlushesWAL(t *testing.T) {
	db, path := openTestDB(t)
	ctx := context.Background()

	w, err := AcquireWriter(db, path, KindSharedSocial, 5e9)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer w.Release()

	tx, err := w.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO t VALUES (1, 'alpha')`); err != nil {
		_ = tx.Rollback()
		t.Fatalf("insert: %v", err)
	}

	if err := w.CommitWithCheckpoint(ctx, tx); err != nil {
		t.Fatalf("CommitWithCheckpoint: %v", err)
	}

	// Close brutal (sans CHECKPOINT additionnel) puis reopen RO.
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	roDB, err := sql.Open("duckdb", path+"?access_mode=READ_ONLY")
	if err != nil {
		t.Fatalf("reopen RO: %v", err)
	}
	defer roDB.Close()
	var got string
	if err := roDB.QueryRow(`SELECT val FROM t WHERE id = 1`).Scan(&got); err != nil {
		t.Fatalf("query: %v", err)
	}
	if got != "alpha" {
		t.Errorf("attendu 'alpha' après CHECKPOINT, got %q", got)
	}
}

// TestCommitWithCheckpoint_NonFatalCheckpointError : si CHECKPOINT échoue
// (ex: lock contention simulé), la fonction loggue WARN mais retourne nil
// (les data sont déjà commit, donc OK fonctionnellement).
//
// NB : on ne peut pas trivialement faire échouer le CHECKPOINT sur une DB
// saine, donc ce test vérifie surtout la non-régression du chemin nominal.
func TestCommitWithCheckpoint_BasicPath(t *testing.T) {
	db, path := openTestDB(t)
	ctx := context.Background()

	w, err := AcquireWriter(db, path, KindSharedSocial, 5e9)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer w.Release()

	tx, err := w.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO t VALUES (2, 'beta')`); err != nil {
		_ = tx.Rollback()
		t.Fatalf("insert: %v", err)
	}
	if err := w.CommitWithCheckpoint(ctx, tx); err != nil {
		t.Errorf("CommitWithCheckpoint nominal devrait retourner nil, got %v", err)
	}
}

// TestCommitWithCheckpoint_CommitFails_PropagatesError : si Commit échoue
// (ex: TX déjà rollbackée), CommitWithCheckpoint doit propager l'erreur.
func TestCommitWithCheckpoint_CommitFails_PropagatesError(t *testing.T) {
	db, path := openTestDB(t)
	ctx := context.Background()

	w, err := AcquireWriter(db, path, KindSharedSocial, 5e9)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer w.Release()

	tx, err := w.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	// Rollback explicit avant CommitWithCheckpoint → commit doit échouer.
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	err = w.CommitWithCheckpoint(ctx, tx)
	if err == nil {
		t.Error("CommitWithCheckpoint sur TX rollbackée devrait erreur")
	}
}

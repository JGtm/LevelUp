package sync

// Tests unitaires PURS de SharedAccess (aucune DB : closures fake + db nil —
// assertSharedWritable(nil) est no-op par contrat best-effort).

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"levelup/go-api/internal/ctxkeys"
)

// TestSharedAccess_PinnedPassthrough : pinned = handle tenu retourné tel quel,
// release no-op (byte-identique à l'historique V1 non-batch).
func TestSharedAccess_PinnedPassthrough(t *testing.T) {
	a := NewPinnedSharedAccess(nil) // nil : probe best-effort no-op
	if _, release, err := a.Read(context.Background()); err != nil {
		t.Fatalf("Read pinned: %v", err)
	} else {
		release()
	}
	if _, release, err := a.Write(context.Background(), "step"); err != nil {
		t.Fatalf("Write pinned: %v", err)
	} else {
		release()
	}
}

// TestSharedAccess_WriteLabelsStep : le burst Write porte le label composé
// "<label>/<step>" via ctxkeys (ventilation par-détenteur étape 0) et
// n'appelle l'acquirer qu'à la demande (paresseux).
func TestSharedAccess_WriteLabelsStep(t *testing.T) {
	var acquires int
	var seenLabel string
	a := NewBurstSharedAccess(
		func(ctx context.Context) (*sql.DB, func(), error) {
			acquires++
			seenLabel = ctxkeys.DBWriterLabel(ctx)
			return testSharedAccessDB(t), func() {}, nil
		},
		nil, "sync_v2_postsync")

	if acquires != 0 {
		t.Fatal("l'acquirer ne doit être appelé qu'au 1er Write (paresseux)")
	}
	_, release, err := a.Write(context.Background(), "events")
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	release()
	if acquires != 1 {
		t.Errorf("acquires = %d, want 1", acquires)
	}
	if seenLabel != "sync_v2_postsync/events" {
		t.Errorf("label = %q, want sync_v2_postsync/events", seenLabel)
	}
}

// TestSharedAccess_ReadUsesROPath : Read passe par le lecteur RO (jamais par
// l'acquirer writer) et le release est idempotent (OnceFunc).
func TestSharedAccess_ReadUsesROPath(t *testing.T) {
	var acquires, reads, releases int
	a := NewBurstSharedAccess(
		func(ctx context.Context) (*sql.DB, func(), error) {
			acquires++
			return testSharedAccessDB(t), func() {}, nil
		},
		func(ctx context.Context) (*sql.DB, func(), error) {
			reads++
			return testSharedAccessDB(t), func() { releases++ }, nil
		},
		"sync_v2_postsync")

	_, release, err := a.Read(context.Background())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	release()
	release() // idempotent
	if reads != 1 || acquires != 0 {
		t.Errorf("reads=%d acquires=%d, want 1/0 (lecture = RO, pas writer)", reads, acquires)
	}
	if releases != 1 {
		t.Errorf("releases=%d, want 1 (OnceFunc)", releases)
	}
}

// TestSharedAccess_WriteRefusedDuringRead : garde anti-deadlock — un burst Write
// pendant un Read en vol est refusé ; après release, il passe.
func TestSharedAccess_WriteRefusedDuringRead(t *testing.T) {
	a := NewBurstSharedAccess(
		func(ctx context.Context) (*sql.DB, func(), error) { return testSharedAccessDB(t), func() {}, nil },
		func(ctx context.Context) (*sql.DB, func(), error) { return testSharedAccessDB(t), func() {}, nil },
		"lbl")

	_, releaseRead, err := a.Read(context.Background())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if _, _, werr := a.Write(context.Background(), "step"); werr == nil {
		t.Fatal("Write pendant un Read en vol aurait dû être refusé (anti-deadlock drain)")
	}
	releaseRead()
	if _, release, werr := a.Write(context.Background(), "step"); werr != nil {
		t.Fatalf("Write après release du Read: %v", werr)
	} else {
		release()
	}
}

// TestSharedAccess_LegacyReadFallsBackToWrite : sans lecteur RO (mode legacy),
// Read délègue à un burst writer.
func TestSharedAccess_LegacyReadFallsBackToWrite(t *testing.T) {
	var acquires int
	a := NewBurstSharedAccess(
		func(ctx context.Context) (*sql.DB, func(), error) {
			acquires++
			return testSharedAccessDB(t), func() {}, nil
		},
		nil, "lbl")
	_, release, err := a.Read(context.Background())
	if err != nil {
		t.Fatalf("Read legacy: %v", err)
	}
	release()
	if acquires != 1 {
		t.Errorf("acquires=%d, want 1 (fallback burst)", acquires)
	}
}

// TestSharedAccess_AcquireErrorPropagates : une erreur d'acquisition remonte
// telle quelle (dégradation gracieuse par étape côté pipeline).
func TestSharedAccess_AcquireErrorPropagates(t *testing.T) {
	boom := errors.New("boom")
	a := NewBurstSharedAccess(
		func(ctx context.Context) (*sql.DB, func(), error) { return nil, nil, boom },
		nil, "lbl")
	if _, _, err := a.Write(context.Background(), "s"); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
}

// testSharedAccessDB ouvre une DuckDB in-memory jetable (probe RC-A des bursts —
// un *sql.DB zéro-valeur paniquerait dans QueryRowContext).
func testSharedAccessDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb in-memory: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

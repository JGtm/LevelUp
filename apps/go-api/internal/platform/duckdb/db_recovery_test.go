// Package duckdb_test — db_recovery_test.go : tests d'auto-récupération
// d'une connexion DuckDB invalidée.
//
// Couvre :
//   - IsInvalidatedError : reconnaissance des signatures d'erreur fatales
//     observées en prod (log 2026-05-14).
//   - DB.Reopen : ferme + rouvre avec les mêmes paramètres, sqlDB sous-jacent
//     remplacé, cache process-level préservé.
//   - DB.WithReopenOnInvalidated : exécute fn ; retry une fois si invalidation
//     détectée ; remonte les autres erreurs sans toucher à la connexion.

package duckdb_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/platform/duckdb"
)

// ─── IsInvalidatedError ──────────────────────────────────────────────────────

// TestIsInvalidatedError_RecognizesProductionPatterns vérifie que toutes les
// signatures observées dans le log de prod 2026-05-14 sont détectées.
func TestIsInvalidatedError_RecognizesProductionPatterns(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil err",
			err:  nil,
			want: false,
		},
		{
			name: "log de prod 2026-05-14 (message complet)",
			err: errors.New(
				`NotificationsRepo.List: query: FATAL Error: Failed: database has been invalidated because of a previous fatal error. The database must be restarted prior to being used again.
Original error: "Invalid Input Error: Failed to delete all rows from index. Only deleted 0 out of 1 rows."`,
			),
			want: true,
		},
		{
			name: "signature courte 'database has been invalidated'",
			err:  errors.New("database has been invalidated because of a previous fatal error"),
			want: true,
		},
		{
			name: "signature 'Failed to delete all rows from index' seule",
			err:  errors.New("Invalid Input Error: Failed to delete all rows from index. Only deleted 0 out of 1 rows."),
			want: true,
		},
		{
			name: "signature 'database must be restarted prior'",
			err:  errors.New("The database must be restarted prior to being used again"),
			want: true,
		},
		{
			name: "erreur banale (timeout, contrainte)",
			err:  errors.New("UNIQUE constraint failed: player_notifications.xuid"),
			want: false,
		},
		{
			name: "context canceled",
			err:  context.Canceled,
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := duckdb.IsInvalidatedError(tc.err)
			if got != tc.want {
				t.Errorf("IsInvalidatedError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// ─── DB.Reopen ───────────────────────────────────────────────────────────────

// TestDB_Reopen_RestoresUsableConnection ouvre une DB, l'utilise, Reopen,
// puis re-utilise — la nouvelle connexion doit fonctionner sans surprise.
func TestDB_Reopen_RestoresUsableConnection(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "recovery.duckdb")
	db, err := duckdb.OpenReadWrite(dbPath)
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if _, err := db.Exec(ctx, "CREATE TABLE t (id INTEGER, label VARCHAR)"); err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}
	if _, err := db.Exec(ctx, "INSERT INTO t VALUES (1, 'before')"); err != nil {
		t.Fatalf("INSERT before: %v", err)
	}

	oldSQLDB := db.SQLDb()
	if err := db.Reopen(); err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if db.SQLDb() == oldSQLDB {
		t.Error("Reopen n'a pas remplacé le *sql.DB sous-jacent")
	}

	// Après Reopen, la DB doit retrouver les données (fichier persisté).
	var label string
	if err := db.QueryRow(ctx, "SELECT label FROM t WHERE id = 1").Scan(&label); err != nil {
		t.Fatalf("QueryRow after Reopen: %v", err)
	}
	if label != "before" {
		t.Errorf("data perdue après Reopen: got %q", label)
	}

	// Et on doit pouvoir continuer à écrire.
	if _, err := db.Exec(ctx, "INSERT INTO t VALUES (2, 'after')"); err != nil {
		t.Fatalf("INSERT after Reopen: %v", err)
	}
	var n int
	if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM t").Scan(&n); err != nil {
		t.Fatalf("COUNT after Reopen: %v", err)
	}
	if n != 2 {
		t.Errorf("attendu 2 lignes après Reopen, obtenu %d", n)
	}
}

// TestDB_Reopen_PreservesCacheKey vérifie qu'après Reopen, le cache
// process-level pointe toujours sur la même *DB (donc d'autres callers qui
// font OpenReadWrite avec le même path récupèrent la *DB ré-ouverte).
func TestDB_Reopen_PreservesCacheKey(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "cache_reopen.duckdb")

	db1, err := duckdb.OpenReadWrite(dbPath)
	if err != nil {
		t.Fatalf("OpenReadWrite 1: %v", err)
	}
	defer db1.Close()

	if err := db1.Reopen(); err != nil {
		t.Fatalf("Reopen: %v", err)
	}

	// Un 2e OpenReadWrite sur le même path doit retourner la MÊME *DB
	// (ref-counted, cache préservé).
	db2, err := duckdb.OpenReadWrite(dbPath)
	if err != nil {
		t.Fatalf("OpenReadWrite 2: %v", err)
	}
	defer db2.Close()
	if db1 != db2 {
		t.Error("Reopen a cassé le cache : OpenReadWrite retourne une *DB différente")
	}

	// La 2e *DB doit être utilisable.
	if _, err := db2.Exec(context.Background(), "SELECT 1"); err != nil {
		t.Errorf("query après Reopen via cache shared: %v", err)
	}
}

// TestDB_Reopen_PreservesTimezone ouvre une DB avec timezone explicite,
// Reopen, puis vérifie que le SET TimeZone est ré-appliqué.
func TestDB_Reopen_PreservesTimezone(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tz_reopen.duckdb")
	db, err := duckdb.OpenReadWrite(dbPath, "Europe/Paris")
	if err != nil {
		t.Fatalf("OpenReadWrite(tz): %v", err)
	}
	defer db.Close()

	if err := db.Reopen(); err != nil {
		t.Fatalf("Reopen: %v", err)
	}

	var tz string
	if err := db.QueryRow(context.Background(),
		"SELECT current_setting('TimeZone')").Scan(&tz); err != nil {
		t.Fatalf("query TimeZone after Reopen: %v", err)
	}
	if tz != "Europe/Paris" {
		t.Errorf("timezone perdue après Reopen: got %q, want Europe/Paris", tz)
	}
}

// ─── DB.WithReopenOnInvalidated ──────────────────────────────────────────────

// TestWithReopenOnInvalidated_PassesThroughSuccess vérifie que le wrapper
// est transparent quand fn réussit du premier coup.
func TestWithReopenOnInvalidated_PassesThroughSuccess(t *testing.T) {
	db := openTempDB(t)
	called := 0
	err := db.WithReopenOnInvalidated(func() error {
		called++
		return nil
	})
	if err != nil {
		t.Errorf("attendu nil, obtenu %v", err)
	}
	if called != 1 {
		t.Errorf("fn called %d times, want 1", called)
	}
}

// TestWithReopenOnInvalidated_PassesThroughNonInvalidatedError vérifie que
// les erreurs non liées à l'invalidation NE déclenchent PAS de Reopen
// (on n'a pas envie de reopen sur un context.Canceled).
func TestWithReopenOnInvalidated_PassesThroughNonInvalidatedError(t *testing.T) {
	db := openTempDB(t)
	expected := errors.New("UNIQUE constraint failed")
	called := 0
	err := db.WithReopenOnInvalidated(func() error {
		called++
		return expected
	})
	if !errors.Is(err, expected) {
		t.Errorf("attendu %v, obtenu %v", expected, err)
	}
	if called != 1 {
		t.Errorf("fn called %d times, want 1 (pas de retry sur erreur non-invalidée)", called)
	}
}

// TestWithReopenOnInvalidated_RetriesOnceOnInvalidatedError vérifie le
// comportement clé : si fn renvoie une erreur d'invalidation, le wrapper
// fait Reopen + retry une fois. Si le 2e essai réussit, l'erreur initiale
// est masquée.
func TestWithReopenOnInvalidated_RetriesOnceOnInvalidatedError(t *testing.T) {
	db := openTempDB(t)
	var called int32
	err := db.WithReopenOnInvalidated(func() error {
		n := atomic.AddInt32(&called, 1)
		if n == 1 {
			// Simule l'erreur exacte du log de prod.
			return errors.New(
				`NotificationsRepo.List: query: FATAL Error: Failed: database has been invalidated because of a previous fatal error.`,
			)
		}
		return nil
	})
	if err != nil {
		t.Errorf("attendu nil après retry, obtenu %v", err)
	}
	if got := atomic.LoadInt32(&called); got != 2 {
		t.Errorf("fn called %d times, want 2 (1 erreur + 1 retry)", got)
	}

	// La DB doit toujours répondre normalement après le reopen.
	if _, err := db.Exec(context.Background(), "SELECT 1"); err != nil {
		t.Errorf("query après reopen: %v", err)
	}
}

// TestWithReopenOnInvalidated_DoesNotLoopOnPersistentInvalidation : si fn
// renvoie une erreur d'invalidation 2 fois (le reopen n'a pas résolu),
// l'erreur est remontée au caller — pas de boucle infinie.
func TestWithReopenOnInvalidated_DoesNotLoopOnPersistentInvalidation(t *testing.T) {
	db := openTempDB(t)
	var called int32
	persistent := errors.New("database has been invalidated again")
	err := db.WithReopenOnInvalidated(func() error {
		atomic.AddInt32(&called, 1)
		return persistent
	})
	if err == nil {
		t.Fatal("erreur persistante doit être remontée")
	}
	if !duckdb.IsInvalidatedError(err) {
		t.Errorf("erreur remontée doit toujours être une invalidation, obtenu %v", err)
	}
	if got := atomic.LoadInt32(&called); got != 2 {
		t.Errorf("fn called %d times, want exactly 2 (pas de boucle)", got)
	}
}

// TestDB_Reopen_ReturnsErrorIfFileGone vérifie que Reopen() remonte
// proprement l'erreur si le fichier sous-jacent n'est plus accessible (DB
// supprimée pendant le runtime, disque plein, permissions perdues).
//
// Comportement attendu : on retourne l'erreur, le sqlDB d'origine reste
// non touché (best-effort sur le close de l'ancien — voir le code Reopen).
func TestDB_Reopen_ReturnsErrorIfFileGone(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "vanish.duckdb")
	db, err := duckdb.OpenReadWrite(dbPath)
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	defer db.Close()

	// Insère pour s'assurer que le fichier est créé sur disque.
	if _, err := db.Exec(context.Background(),
		"CREATE TABLE t (id INTEGER); INSERT INTO t VALUES (1)"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Force la fermeture du fichier puis le supprime. Tous les handles
	// duckdb partagés sur cette base sont fermés via Close (refCount).
	_ = db.Close()
	if err := os.Remove(dbPath); err != nil {
		t.Fatalf("os.Remove: %v", err)
	}
	// Et on supprime aussi le répertoire pour faire en sorte que la création
	// d'un nouveau fichier échoue (parent absent).
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("os.RemoveAll: %v", err)
	}

	// Reopen doit échouer : pas de fichier, pas de dossier parent.
	if err := db.Reopen(); err == nil {
		t.Error("Reopen sur path inexistant doit remonter une erreur")
	}
}

// TestWithReopenOnInvalidated_RetryFails_SurfacesError vérifie que si la 2e
// tentative échoue avec une erreur DIFFÉRENTE de l'invalidation (par exemple
// une violation de contrainte), cette erreur est remontée telle quelle au
// caller (et pas l'erreur d'invalidation initiale).
func TestWithReopenOnInvalidated_RetryFails_SurfacesError(t *testing.T) {
	db := openTempDB(t)
	var called int32
	otherErr := errors.New("UNIQUE constraint violation")
	err := db.WithReopenOnInvalidated(func() error {
		n := atomic.AddInt32(&called, 1)
		if n == 1 {
			return errors.New("database has been invalidated")
		}
		return otherErr
	})
	if !errors.Is(err, otherErr) {
		t.Errorf("erreur remontée doit être celle du retry, obtenu %v", err)
	}
	if got := atomic.LoadInt32(&called); got != 2 {
		t.Errorf("fn called %d times, want 2", got)
	}
}

// ─── ExecRecovered / QueryRecovered ─────────────────────────────────────────

// TestExecRecovered_NormalPath vérifie que ExecRecovered fonctionne comme un
// Exec classique quand la connexion est saine.
func TestExecRecovered_NormalPath(t *testing.T) {
	db := openTempDB(t)
	ctx := context.Background()

	if _, err := db.ExecRecovered(ctx, "CREATE TABLE t (id INTEGER)"); err != nil {
		t.Fatalf("ExecRecovered CREATE: %v", err)
	}
	res, err := db.ExecRecovered(ctx, "INSERT INTO t VALUES (1), (2), (3)")
	if err != nil {
		t.Fatalf("ExecRecovered INSERT: %v", err)
	}
	if n, _ := res.RowsAffected(); n != 3 {
		t.Errorf("RowsAffected: attendu 3, obtenu %d", n)
	}
}

// TestQueryRecovered_NormalPath vérifie que QueryRecovered itère correctement
// sur les résultats quand la connexion est saine.
func TestQueryRecovered_NormalPath(t *testing.T) {
	db := openTempDB(t)
	ctx := context.Background()
	if _, err := db.ExecRecovered(ctx,
		"CREATE TABLE t (id INTEGER); INSERT INTO t VALUES (1), (2), (3)"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	rows, err := db.QueryRecovered(ctx, "SELECT id FROM t ORDER BY id")
	if err != nil {
		t.Fatalf("QueryRecovered: %v", err)
	}
	defer rows.Close()

	var got []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, id)
	}
	if len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Errorf("got %v, want [1 2 3]", got)
	}
}

// TestExecRecovered_PropagatesNonInvalidatedError vérifie qu'une erreur SQL
// classique (contrainte violée, syntaxe) est remontée telle quelle sans
// déclencher de Reopen inutile.
func TestExecRecovered_PropagatesNonInvalidatedError(t *testing.T) {
	db := openTempDB(t)
	ctx := context.Background()
	if _, err := db.ExecRecovered(ctx,
		"CREATE TABLE t (id INTEGER PRIMARY KEY); INSERT INTO t VALUES (1)"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	oldSQLDB := db.SQLDb()

	_, err := db.ExecRecovered(ctx, "INSERT INTO t VALUES (1)")
	if err == nil {
		t.Fatal("attendu une erreur de contrainte PK")
	}
	if duckdb.IsInvalidatedError(err) {
		t.Errorf("erreur PK ne doit pas être classée invalidation: %v", err)
	}
	if db.SQLDb() != oldSQLDB {
		t.Error("ExecRecovered a fait un Reopen inutile sur erreur non-invalidation")
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func openTempDB(t *testing.T) *duckdb.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, fmt.Sprintf("recovery_%s.duckdb", t.Name()))
	db, err := duckdb.OpenReadWrite(dbPath)
	if err != nil {
		t.Fatalf("openTempDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
